package store

import (
	"bytes"
	"database/sql"
	"errors"
	"math"
	"os"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA/core"
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
	CreateSideChainTxsTable = `CREATE TABLE IF NOT EXISTS SideChainTxs (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR,
				GenesisBlockAddress VARCHAR(34)
			);`
	CreateMainChainTxsTable = `CREATE TABLE IF NOT EXISTS MainChainTxs (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR
			);`
)

var (
	DbCache DataStore
)

type AddressUTXO struct {
	Input               *Input
	Amount              *Fixed64
	GenesisBlockAddress string
	DestroyAddress      string
}

type DataStore interface {
	CurrentHeight(height uint32) uint32
	CurrentSideHeight(genesisBlockAddress string, height uint32) uint32

	AddAddressUTXO(utxo *AddressUTXO) error
	DeleteUTXO(input *Input) error
	GetAddressUTXOsFromGenesisBlockAddress(genesisBlockAddress string) ([]*AddressUTXO, error)
	GetAddressUTXOsFromDestroyAddress(destroyAddress string) ([]*AddressUTXO, error)

	AddSideChainTx(transactionHash, genesisBlockAddress string) error
	HasSideChainTx(transactionHash string) (bool, error)
	RemoveSideChainTxs(transactionHashes []string) error
	GetAllSideChainTxs(genesisBlockAddress string) ([]string, error)

	AddMainChainTx(transactionHash string) error
	HashMainChainTx(transactionHash string) (bool, error)
	RemoveMainChainTxs(transactionHashes []string) error
	GetAllMainChainTxs() ([]string, error)

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
	// Create SideChainTxs table
	_, err = db.Exec(CreateSideChainTxsTable)
	if err != nil {
		return nil, err
	}
	// Create MainChainTxs table
	_, err = db.Exec(CreateMainChainTxsTable)
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

func (store *DataStoreImpl) DeleteUTXO(input *Input) error {
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

		var input Input
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

		var input Input
		reader := bytes.NewReader(outputBytes)
		input.Deserialize(reader)

		var amount Fixed64
		reader = bytes.NewReader(amountBytes)
		amount.Deserialize(reader)

		inputs = append(inputs, &AddressUTXO{&input, &amount, genesisBlockAddress, destroyAddress})
	}
	return inputs, nil
}

func (store *DataStoreImpl) AddSideChainTx(transactionHash, genesisBlockAddress string) error {
	store.sideMux.Lock()
	defer store.sideMux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO SideChainTxs(TransactionHash, GenesisBlockAddress) values(?,?)")
	if err != nil {
		return err
	}
	// Do insert
	_, err = stmt.Exec(transactionHash, genesisBlockAddress)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) HasSideChainTx(transactionHash string) (bool, error) {
	store.sideMux.Lock()
	defer store.sideMux.Unlock()

	rows, err := store.Query(`SELECT GenesisBlockAddress FROM SideChainTxs WHERE TransactionHash=?`, transactionHash)
	defer rows.Close()
	if err != nil {
		return false, err
	}

	return rows.Next(), nil
}

func (store *DataStoreImpl) RemoveSideChainTxs(transactionHashes []string) error {
	store.sideMux.Lock()
	defer store.sideMux.Unlock()

	for _, txHash := range transactionHashes {
		stmt, err := store.Prepare(
			"DELETE FROM SideChainTxs WHERE TransactionHash=?")
		if err != nil {
			return err
		}
		_, err = stmt.Exec(txHash)
		if err != nil {
			return err
		}
	}

	return nil
}

func (store *DataStoreImpl) GetAllSideChainTxs(genesisBlockAddress string) ([]string, error) {
	store.sideMux.Lock()
	defer store.sideMux.Unlock()

	rows, err := store.Query(`SELECT SideChainTxs.TransactionHash FROM SideChainTxs WHERE GenesisBlockAddress=?`, genesisBlockAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txHashes []string
	for rows.Next() {
		var txHash string
		err = rows.Scan(&txHash)
		if err != nil {
			return nil, err
		}
		txHashes = append(txHashes, txHash)
	}
	return txHashes, nil
}

func (store *DataStoreImpl) AddMainChainTx(transactionHash string) error {
	store.mainMux.Lock()
	defer store.mainMux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO MainChainTxs(TransactionHash) values(?)")
	if err != nil {
		return err
	}
	// Do insert
	_, err = stmt.Exec(transactionHash)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) HashMainChainTx(transactionHash string) (bool, error) {
	store.mainMux.Lock()
	defer store.mainMux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash FROM MainChainTxs WHERE TransactionHash=?`, transactionHash)
	defer rows.Close()
	if err != nil {
		return false, err
	}

	return rows.Next(), nil
}

func (store *DataStoreImpl) RemoveMainChainTxs(transactionHashes []string) error {
	store.mainMux.Lock()
	defer store.mainMux.Unlock()

	for _, txHash := range transactionHashes {
		stmt, err := store.Prepare(
			"DELETE FROM MainChainTxs WHERE TransactionHash=?")
		if err != nil {
			return err
		}
		_, err = stmt.Exec(txHash)
		if err != nil {
			return err
		}
	}

	return nil
}

func (store *DataStoreImpl) GetAllMainChainTxs() ([]string, error) {
	store.mainMux.Lock()
	defer store.mainMux.Unlock()

	rows, err := store.Query(`SELECT MainChainTxs.TransactionHash FROM MainChainTxs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txHashes []string
	for rows.Next() {
		var txHash string
		err = rows.Scan(&txHash)
		if err != nil {
			return nil, err
		}
		txHashes = append(txHashes, txHash)
	}
	return txHashes, nil
}

type DbMainChainFunc struct {
}

func (dbFunc *DbMainChainFunc) GetAvailableUtxos(withdrawBank string) ([]*AddressUTXO, error) {
	utxos, err := DbCache.GetAddressUTXOsFromGenesisBlockAddress(withdrawBank)
	if err != nil {
		return nil, errors.New("Get spender's UTXOs failed.")
	}
	var availableUTXOs []*AddressUTXO
	var currentHeight = DbCache.CurrentHeight(QueryHeightCode)
	for _, utxo := range utxos {
		if utxo.Input.Sequence > 0 {
			if utxo.Input.Sequence >= currentHeight {
				continue
			}
			utxo.Input.Sequence = math.MaxUint32 - 1
		}
		availableUTXOs = append(availableUTXOs, utxo)
	}
	availableUTXOs = SortUTXOs(availableUTXOs)

	return availableUTXOs, nil
}

func (dbFunc *DbMainChainFunc) GetMainNodeCurrentHeight() (uint32, error) {
	chainHeight, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return 0, err
	}
	return chainHeight, nil
}
