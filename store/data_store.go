package store

import (
	"bytes"
	"database/sql"
	"math"
	"os"
	"sync"

	. "github.com/elastos/Elastos.ELA.Arbiter/common"
	"github.com/elastos/Elastos.ELA.Arbiter/common/config"
	"github.com/elastos/Elastos.ELA.Arbiter/common/log"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DriverName      = "sqlite3"
	DBName          = "./chainUTXOCache.db"
	QueryHeightCode = 0
	ResetHeightCode = math.MaxUint32
)

const (
	CreateInfoTable = `CREATE TABLE IF NOT EXISTS Info (
				Name VARCHAR(20) NOT NULL PRIMARY KEY,
				Value BLOB
			);`
	CreateHeightInfoTable = `CREATE TABLE IF NOT EXISTS SideHeightInfo (
				GenesisBlockAddress VARCHAR(34) NOT NULL PRIMARY KEY,
				Height INTEGER 
			);`
	CreateUTXOsTable = `CREATE TABLE IF NOT EXISTS UTXOs (
				Id INTEGER NOT NULL PRIMARY KEY,
				UTXOInput BLOB UNIQUE,
				Amount VARCHAR,
				GenesisBlockAddress VARCHAR(34),
				DestroyAddress VARCHAR(34)
			);`
	CreateSideChainMiningTable = `CREATE TABLE IF NOT EXISTS SideChainMining (
				GenesisBlockAddress VARCHAR(34) NOT NULL PRIMARY KEY,
				MainHeight INTEGER,
				SideHeight INTEGER,
				Offset INTEGER
			);`
)

var (
	DbCache DataStore
)

type AddressUTXO struct {
	Input               *tx.UTXOTxInput
	Amount              *Fixed64
	GenesisBlockAddress string
	DestroyAddress      string
}

type DataStore interface {
	CurrentHeight(height uint32) uint32
	CurrentSideHeight(genesisBlockAddress string, height uint32) uint32

	AddAddressUTXO(utxo *AddressUTXO) error
	DeleteUTXO(input *tx.UTXOTxInput) error
	GetAddressUTXOsFromGenesisBlockAddress(genesisBlockAddress string) ([]*AddressUTXO, error)
	GetAddressUTXOsFromDestroyAddress(destroyAddress string) ([]*AddressUTXO, error)

	SetMiningRecord(genesisBlockAddress string, mainHeight uint32, sideHeight uint32, offset uint8) error
	GetMiningRecord(genesisBlockAddress string, mainHeight *uint32, sideHeight *uint32, offset *uint8) (bool, error)

	ResetDataStore() error
}

type DataStoreImpl struct {
	mainMux   *sync.Mutex
	sideMux   *sync.Mutex
	miningMux *sync.Mutex

	*sql.DB
}

func OpenDataStore() (DataStore, error) {
	db, err := initDB()
	if err != nil {
		return nil, err
	}
	dataStore := &DataStoreImpl{DB: db, mainMux: new(sync.Mutex), sideMux: new(sync.Mutex), miningMux: new(sync.Mutex)}

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
	// Create SideChainMining table
	_, err = db.Exec(CreateSideChainMiningTable)
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
		store.mainMux.Lock()
		store.sideMux.Lock()
		store.miningMux.Lock()
		store.Close()
		os.Exit(-1)
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
	store.mainMux.Lock()
	defer store.mainMux.Unlock()

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
	store.sideMux.Lock()
	defer store.sideMux.Unlock()

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
	store.mainMux.Lock()
	defer store.mainMux.Unlock()

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
	store.mainMux.Lock()
	defer store.mainMux.Unlock()

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
	store.mainMux.Lock()
	defer store.mainMux.Unlock()

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
	store.mainMux.Lock()
	defer store.mainMux.Unlock()

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
		err = rows.Scan(&outputBytes, &amountBytes, &destroyAddress)
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

func (store *DataStoreImpl) SetMiningRecord(genesisBlockAddress string, mainHeight uint32, sideHeight uint32, offset uint8) error {
	store.miningMux.Lock()
	defer store.miningMux.Unlock()

	rows, err := store.Query(`SELECT * FROM SideChainMining WHERE GenesisBlockAddress=?`, genesisBlockAddress)
	if err != nil {
		return err
	}

	if rows.Next() {
		err = rows.Close()
		if err != nil {
			return err
		}

		stmt, err := store.Prepare("UPDATE SideChainMining SET MainHeight=?, SideHeight=?, Offset=? WHERE GenesisBlockAddress=?")
		if err != nil {
			return err
		}
		_, err = stmt.Exec(mainHeight, sideHeight, offset, genesisBlockAddress)
		if err != nil {
			return err
		}

	} else {
		rows.Close()

		// Prepare sql statement
		stmt, err := store.Prepare("INSERT INTO SideChainMining(GenesisBlockAddress, MainHeight, SideHeight, Offset) values(?,?,?,?)")
		if err != nil {
			return err
		}
		// Do insert
		_, err = stmt.Exec(genesisBlockAddress, mainHeight, sideHeight, offset)
		if err != nil {
			return err
		}
	}
	return nil
}

func (store *DataStoreImpl) GetMiningRecord(genesisBlockAddress string, mainHeight *uint32, sideHeight *uint32, offset *uint8) (bool, error) {
	store.miningMux.Lock()
	defer store.miningMux.Unlock()

	rows, err := store.Query(`SELECT MainHeight, SideHeight, Offset FROM SideChainMining WHERE GenesisBlockAddress=?`, genesisBlockAddress)
	defer rows.Close()
	if err != nil {
		return false, err
	}

	if rows.Next() {
		err = rows.Scan(mainHeight, sideHeight, offset)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}
