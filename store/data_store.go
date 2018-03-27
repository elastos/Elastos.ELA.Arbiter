package store

import (
	"bytes"
	"database/sql"
	"math"
	"os"
	"sync"

	. "Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/common/log"
	tx "Elastos.ELA.Arbiter/core/transaction"

	"Elastos.ELA.Arbiter/common/config"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DriverName      = "sqlite3"
	DBName          = "./chainUTXOCache.db"
	QueryHeightCode = 0
	ResetHeightCode = math.MaxUint32
)

const (
	//TODO set all addresses with fix width varchar.
	CreateInfoTable = `CREATE TABLE IF NOT EXISTS Info (
				Name VARCHAR(20) NOT NULL PRIMARY KEY,
				Value BLOB
			);`
	CreateHeightInfoTable = `CREATE TABLE IF NOT EXISTS SideHeightInfo (
				GenesisBlockAddress VARCHAR NOT NULL PRIMARY KEY,
				Height INTEGER 
			);`
	CreateUTXOsTable = `CREATE TABLE IF NOT EXISTS UTXOs (
				Id INTEGER NOT NULL PRIMARY KEY,
				UTXOInput BLOB UNIQUE,
				Amount VARCHAR,
				GenesisBlockAddress VARCHAR,
				DestroyAddress VARCHAR
			);`
)

var (
	DB DataStore
)

type AddressUTXO struct {
	Input               *tx.UTXOTxInput
	Amount              *Fixed64
	GenesisBlockAddress string
	DestroyAddress      string
}

type DataStore interface {
	sync.Locker

	CurrentHeight(height uint32) uint32
	CurrentSideHeight(genesisBlockAddress string, height uint32) uint32

	AddAddressUTXO(utxo *AddressUTXO) error
	DeleteUTXO(input *tx.UTXOTxInput) error
	GetAddressUTXOsFromGenesisBlockAddress(genesisBlockAddress string) ([]*AddressUTXO, error)
	GetAddressUTXOsFromDestroyAddress(destroyAddress string) ([]*AddressUTXO, error)

	ResetDataStore() error
}

type DataStoreImpl struct {
	sync.Mutex

	*sql.DB
}

func OpenDataStore() (DataStore, error) {
	db, err := initDB()
	if err != nil {
		return nil, err
	}
	dataStore := &DataStoreImpl{DB: db}

	// Handle system interrupt signals
	dataStore.catchSystemSignals()

	return dataStore, nil
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open(DriverName, DBName)
	if err != nil {
		log.Error("Open data db error:", err)
		return nil, err
	}
	// Create info table
	_, err = db.Exec(CreateInfoTable)
	if err != nil {
		return nil, err
	}
	// Create SideHeightInfo table
	_, err = db.Exec(CreateHeightInfoTable)
	if err != nil {
		return nil, err
	}
	// Create UTXOs table
	_, err = db.Exec(CreateUTXOsTable)
	if err != nil {
		return nil, err
	}

	stmt, err := db.Prepare("INSERT INTO Info(Name, Value) values(?,?)")
	if err != nil {
		return nil, err
	}
	stmt.Exec("Height", uint32(0))

	for _, node := range config.Parameters.SideNodeList {
		stmt, err := db.Prepare("INSERT INTO SideHeightInfo(GenesisBlockAddress, Height) values(?,?)")
		if err != nil {
			return nil, err
		}
		stmt.Exec(node.GenesisBlockAddress, uint32(0))
	}

	return db, nil
}

func (store *DataStoreImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.Lock()
		store.Close()
	})
}

func (store *DataStoreImpl) ResetDataStore() error {

	store.DB.Close()
	os.Remove(DBName)

	var err error
	store.DB, err = initDB()
	if err != nil {
		return err
	}

	return nil
}

func (store *DataStoreImpl) CurrentHeight(height uint32) uint32 {
	store.Lock()
	defer store.Unlock()

	row := store.QueryRow("SELECT Value FROM Info WHERE Name=?", "Height")
	var storedHeight uint32
	row.Scan(&storedHeight)

	if height > storedHeight {
		// Received reset height code
		if height == ResetHeightCode {
			height = 0
		}
		// Insert current height
		stmt, err := store.Prepare("UPDATE Info SET Value=? WHERE Name=?")
		if err != nil {
			return uint32(0)
		}
		_, err = stmt.Exec(height, "Height")
		if err != nil {
			return uint32(0)
		}
		return height
	}
	return storedHeight
}

func (store *DataStoreImpl) CurrentSideHeight(genesisBlockAddress string, height uint32) uint32 {
	store.Lock()
	defer store.Unlock()

	row := store.QueryRow("SELECT Height FROM SideHeightInfo WHERE GenesisBlockAddress=?", genesisBlockAddress)
	var storedHeight uint32
	row.Scan(&storedHeight)

	if height > storedHeight {
		// Received reset height code
		if height == ResetHeightCode {
			height = 0
		}
		// Insert current height
		stmt, err := store.Prepare("UPDATE SideHeightInfo SET Height=? WHERE GenesisBlockAddress=?")
		if err != nil {
			return uint32(0)
		}
		_, err = stmt.Exec(height, genesisBlockAddress)
		if err != nil {
			return uint32(0)
		}
		return height
	}
	return storedHeight
}

func (store *DataStoreImpl) AddAddressUTXO(utxo *AddressUTXO) error {
	store.Lock()
	defer store.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO UTXOs(UTXOInput, Amount, GenesisBlockAddress, DestroyAddress) values(?,?,?,?)")
	if err != nil {
		return err
	}
	// Serialize input
	buf := new(bytes.Buffer)
	utxo.Input.Serialize(buf)
	inputBytes := buf.Bytes()
	// Serialize amount
	buf = new(bytes.Buffer)
	utxo.Amount.Serialize(buf)
	amountBytes := buf.Bytes()
	// Do insert
	_, err = stmt.Exec(inputBytes, amountBytes, utxo.GenesisBlockAddress, utxo.DestroyAddress)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) DeleteUTXO(input *tx.UTXOTxInput) error {
	store.Lock()
	defer store.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("DELETE FROM UTXOs WHERE UTXOInput=?")
	if err != nil {
		return err
	}
	// Serialize input
	buf := new(bytes.Buffer)
	input.Serialize(buf)
	inputBytes := buf.Bytes()
	// Do delete
	_, err = stmt.Exec(inputBytes)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) GetAddressUTXOsFromDestroyAddress(destroyAddress string) ([]*AddressUTXO, error) {
	store.Lock()
	defer store.Unlock()

	rows, err := store.Query(`SELECT UTXOs.UTXOInput, UTXOs.Amount, UTXOs.GenesisBlockAddress FROM UTXOs WHERE DestroyAddress=?`, destroyAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inputs []*AddressUTXO
	for rows.Next() {
		var outputBytes []byte
		var amountBytes []byte
		var genesisBlockAddress string
		err = rows.Scan(&outputBytes, &amountBytes, &genesisBlockAddress)
		if err != nil {
			return nil, err
		}

		var input tx.UTXOTxInput
		reader := bytes.NewReader(outputBytes)
		input.Deserialize(reader)

		var amount Fixed64
		reader = bytes.NewReader(amountBytes)
		amount.Deserialize(reader)

		inputs = append(inputs, &AddressUTXO{&input, &amount, genesisBlockAddress, destroyAddress})
	}
	return inputs, nil
}

func (store *DataStoreImpl) GetAddressUTXOsFromGenesisBlockAddress(genesisBlockAddress string) ([]*AddressUTXO, error) {
	store.Lock()
	defer store.Unlock()

	rows, err := store.Query(`SELECT UTXOs.UTXOInput, UTXOs.Amount, UTXOs.DestroyAddress FROM UTXOs WHERE GenesisBlockAddress=?`, genesisBlockAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inputs []*AddressUTXO
	for rows.Next() {
		var outputBytes []byte
		var amountBytes []byte
		var destroyAddress string
		err = rows.Scan(&outputBytes, &amountBytes, &genesisBlockAddress, &destroyAddress)
		if err != nil {
			return nil, err
		}

		var input tx.UTXOTxInput
		reader := bytes.NewReader(outputBytes)
		input.Deserialize(reader)

		var amount Fixed64
		reader = bytes.NewReader(amountBytes)
		amount.Deserialize(reader)

		inputs = append(inputs, &AddressUTXO{&input, &amount, genesisBlockAddress, destroyAddress})
	}
	return inputs, nil
}
