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
	"github.com/elastos/Elastos.ELA/bloom"
	. "github.com/elastos/Elastos.ELA/core"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DriverName      = "sqlite3"
	DBDocumentNAME  = "./DBCache"
	DBNameUTXO      = "./DBCache/chainUTXOCache.db"
	DBNameMainChain = "./DBCache/mainChainCache.db"
	DBNameSideChain = "./DBCache/sideChainCache.db"
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
				GenesisBlockAddress VARCHAR(34)
			);`
	CreateSideChainTxsTable = `CREATE TABLE IF NOT EXISTS SideChainTxs (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR,
				GenesisBlockAddress VARCHAR(34),
				TransactionData BLOB,
				BlockHeight INTEGER 
			);`
	CreateMainChainTxsTable = `CREATE TABLE IF NOT EXISTS MainChainTxs (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR,
				GenesisBlockAddress VARCHAR(34),
				TransactionData BLOB,
				MerkleProof BLOB
			);`
)

var (
	DbCache DataStoreImpl
)

type AddressUTXO struct {
	Input               *Input
	Amount              *Fixed64
	GenesisBlockAddress string
}

type DataStore interface {
	ResetDataStore() error
	catchSystemSignals()
}

type DataStoreUTXO interface {
	DataStore

	CurrentHeight(height uint32) uint32
	AddAddressUTXO(utxo *AddressUTXO) error
	DeleteUTXO(input *Input) error
	GetAddressUTXOsFromGenesisBlockAddress(genesisBlockAddress string) ([]*AddressUTXO, error)
}

type DataStoreMainChain interface {
	DataStore

	AddMainChainTx(transactionHash string, genesisBlockAddress string, transaction *Transaction, proof *bloom.MerkleProof) error
	HasMainChainTx(transactionHash string) (bool, error)
	RemoveMainChainTx(transactionHash, genesisBlockAddress string) error
	RemoveMainChainTxs(transactionHashes, genesisBlockAddress []string) error
	GetAllMainChainTxHashes() ([]string, []string, error)
	GetAllMainChainTxs() ([]string, []string, []*Transaction, []*bloom.MerkleProof, error)
	GetMainChainTxsFromHashes(transactionHashes []string, genesisBlockAddresses string) ([]*Transaction, []*bloom.MerkleProof, error)
}

type DataStoreSideChain interface {
	DataStore

	CurrentSideHeight(genesisBlockAddress string, height uint32) uint32
	AddSideChainTx(transactionHash, genesisBlockAddress string, transaction *Transaction, blockHeight uint32) error
	AddSideChainTxs(transactionHashes, genesisBlockAddresses []string, transactionsBytes [][]byte, blockHeights []uint32) error
	HasSideChainTx(transactionHash string) (bool, error)
	RemoveSideChainTxs(transactionHashes []string) error
	GetAllSideChainTxHashes() ([]string, error)
	GetAllSideChainTxHashesAndHeights(genesisBlockAddress string) ([]string, []uint32, error)
	GetSideChainTxsFromHashes(transactionHashes []string) ([]*Transaction, error)
	GetSideChainTxsFromHashesAndGenesisAddress(transactionHashes []string, genesisBlockAddress string) ([]*Transaction, error)
}

type DataStoreImpl struct {
	UTXOStore      DataStoreUTXOImpl
	MainChainStore DataStoreMainChainImpl
	SideChainStore DataStoreSideChainImpl
}

type DataStoreUTXOImpl struct {
	mux *sync.Mutex

	*sql.DB
}

type DataStoreMainChainImpl struct {
	mux *sync.Mutex

	*sql.DB
}

type DataStoreSideChainImpl struct {
	mux *sync.Mutex

	*sql.DB
}

func OpenDataStore() (*DataStoreImpl, error) {
	dbUTXO, err := initUTXODB()
	if err != nil {
		return nil, err
	}
	dbMainChain, err := initMainChainDB()
	if err != nil {
		return nil, err
	}
	dbSideChain, err := initSideChainDB()
	if err != nil {
		return nil, err
	}
	dataStore := &DataStoreImpl{
		UTXOStore:      DataStoreUTXOImpl{mux: new(sync.Mutex), DB: dbUTXO},
		MainChainStore: DataStoreMainChainImpl{mux: new(sync.Mutex), DB: dbMainChain},
		SideChainStore: DataStoreSideChainImpl{mux: new(sync.Mutex), DB: dbSideChain}}

	// Handle system interrupt signals
	dataStore.UTXOStore.catchSystemSignals()
	dataStore.MainChainStore.catchSystemSignals()
	dataStore.SideChainStore.catchSystemSignals()

	return dataStore, nil
}

func OpenUTXODataStore() (*DataStoreUTXOImpl, error) {
	dbUTXO, err := initUTXODB()
	if err != nil {
		return nil, err
	}
	dataStore := &DataStoreUTXOImpl{mux: new(sync.Mutex), DB: dbUTXO}

	// Handle system interrupt signals
	dataStore.catchSystemSignals()

	return dataStore, nil
}

func OpenMainChainDataStore() (*DataStoreMainChainImpl, error) {
	dbMainChain, err := initMainChainDB()
	if err != nil {
		return nil, err
	}
	dataStore := &DataStoreMainChainImpl{mux: new(sync.Mutex), DB: dbMainChain}

	// Handle system interrupt signals
	dataStore.catchSystemSignals()

	return dataStore, nil
}

func OpenSideChainDataStore() (*DataStoreSideChainImpl, error) {
	dbSideChain, err := initSideChainDB()
	if err != nil {
		return nil, err
	}
	dataStore := &DataStoreSideChainImpl{mux: new(sync.Mutex), DB: dbSideChain}

	// Handle system interrupt signals
	dataStore.catchSystemSignals()

	return dataStore, nil
}

func initUTXODB() (*sql.DB, error) {
	err := CheckAndCreateDocument(DBDocumentNAME)
	if err != nil {
		log.Error("Create DBCache doucument error:", err)
		return nil, err
	}
	db, err := sql.Open(DriverName, DBNameUTXO)
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
	return db, nil
}

func initMainChainDB() (*sql.DB, error) {
	err := CheckAndCreateDocument(DBDocumentNAME)
	if err != nil {
		log.Error("Create DBCache doucument error:", err)
		return nil, err
	}
	db, err := sql.Open(DriverName, DBNameMainChain)
	if err != nil {
		log.Error("Open data db error:", err)
		return nil, err
	}
	// Create MainChainTxs table
	_, err = db.Exec(CreateMainChainTxsTable)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func initSideChainDB() (*sql.DB, error) {
	err := CheckAndCreateDocument(DBDocumentNAME)
	if err != nil {
		log.Error("Create DBCache doucument error:", err)
		return nil, err
	}
	db, err := sql.Open(DriverName, DBNameSideChain)
	if err != nil {
		log.Error("Open data db error:", err)
		return nil, err
	}
	// Create SideHeightInfo table
	_, err = db.Exec(CreateHeightInfoTable)
	if err != nil {
		return nil, err
	}
	// Create SideChainTxs table
	_, err = db.Exec(CreateSideChainTxsTable)
	if err != nil {
		return nil, err
	}

	for _, node := range config.Parameters.SideNodeList {
		stmt, err := db.Prepare("INSERT INTO SideHeightInfo(GenesisBlockAddress, Height) values(?,?)")
		if err != nil {
			return nil, err
		}
		stmt.Exec(node.GenesisBlockAddress, uint32(0))
	}

	return db, nil
}

func (store *DataStoreUTXOImpl) ResetDataStore() error {
	store.DB.Close()
	os.Remove(DBNameUTXO)

	var err error
	store.DB, err = initUTXODB()
	if err != nil {
		return err
	}

	return nil
}

func (store *DataStoreUTXOImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.mux.Lock()
		store.DB.Close()
		os.Exit(-1)
	})
}

func (store *DataStoreUTXOImpl) CurrentHeight(height uint32) uint32 {
	store.mux.Lock()
	defer store.mux.Unlock()

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

func (store *DataStoreUTXOImpl) AddAddressUTXO(utxo *AddressUTXO) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO UTXOs(UTXOInput, Amount, GenesisBlockAddress) values(?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Serialize input
	buf := new(bytes.Buffer)
	utxo.Input.Serialize(buf)
	inputBytes := buf.Bytes()
	// Serialize amount
	buf = new(bytes.Buffer)
	utxo.Amount.Serialize(buf)
	amountBytes := buf.Bytes()
	// Do insert
	_, err = stmt.Exec(inputBytes, amountBytes, utxo.GenesisBlockAddress)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreUTXOImpl) DeleteUTXO(input *Input) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("DELETE FROM UTXOs WHERE UTXOInput=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

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

func (store *DataStoreUTXOImpl) GetAddressUTXOsFromGenesisBlockAddress(genesisBlockAddress string) ([]*AddressUTXO, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT UTXOs.UTXOInput, UTXOs.Amount FROM UTXOs WHERE GenesisBlockAddress=?`, genesisBlockAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inputs []*AddressUTXO
	for rows.Next() {
		var outputBytes []byte
		var amountBytes []byte
		err = rows.Scan(&outputBytes, &amountBytes)
		if err != nil {
			return nil, err
		}

		var input Input
		reader := bytes.NewReader(outputBytes)
		input.Deserialize(reader)

		var amount Fixed64
		reader = bytes.NewReader(amountBytes)
		amount.Deserialize(reader)

		inputs = append(inputs, &AddressUTXO{&input, &amount, genesisBlockAddress})
	}
	return inputs, nil
}

func (store *DataStoreSideChainImpl) ResetDataStore() error {
	store.DB.Close()
	os.Remove(DBNameSideChain)

	var err error
	store.DB, err = initSideChainDB()
	if err != nil {
		return err
	}

	return nil
}

func (store *DataStoreSideChainImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.mux.Lock()
		store.DB.Close()
		os.Exit(-1)
	})
}

func (store *DataStoreSideChainImpl) CurrentSideHeight(genesisBlockAddress string, height uint32) uint32 {
	store.mux.Lock()
	defer store.mux.Unlock()

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

func (store *DataStoreSideChainImpl) AddSideChainTxs(transactionHashes, genesisBlockAddresses []string, transactionsBytes [][]byte, blockHeights []uint32) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}

	// Prepare sql statement
	stmt, err := tx.Prepare("INSERT INTO SideChainTxs(TransactionHash, GenesisBlockAddress, TransactionData, BlockHeight) values(?,?,?,?)")
	if err != nil {
		return err
	}

	// Do insert
	for i := 0; i < len(transactionHashes); i++ {
		_, err = stmt.Exec(transactionHashes[i], genesisBlockAddresses[i], transactionsBytes[i], blockHeights[i])
		if err != nil {
			return err
		}
	}
	stmt.Close()
	tx.Commit()

	return nil
}

func (store *DataStoreSideChainImpl) AddSideChainTx(transactionHash, genesisBlockAddress string,
	transaction *Transaction, blockHeight uint32) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO SideChainTxs(TransactionHash, GenesisBlockAddress, TransactionData, BlockHeight) values(?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Serialize transaction
	buf := new(bytes.Buffer)
	transaction.Serialize(buf)
	transactionBytes := buf.Bytes()

	// Do insert
	_, err = stmt.Exec(transactionHash, genesisBlockAddress, transactionBytes, blockHeight)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreSideChainImpl) HasSideChainTx(transactionHash string) (bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT GenesisBlockAddress FROM SideChainTxs WHERE TransactionHash=?`, transactionHash)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (store *DataStoreSideChainImpl) RemoveSideChainTxs(transactionHashes []string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("DELETE FROM SideChainTxs WHERE TransactionHash=?")
	if err != nil {
		return err
	}

	for _, txHash := range transactionHashes {
		stmt.Exec(txHash)
	}
	stmt.Close()
	tx.Commit()

	return nil
}

func (store *DataStoreSideChainImpl) GetAllSideChainTxHashes() ([]string, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT SideChainTxs.TransactionHash FROM SideChainTxs GROUP BY TransactionHash`)
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

func (store *DataStoreSideChainImpl) GetAllSideChainTxHashesAndHeights(genesisBlockAddress string) ([]string, []uint32, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT SideChainTxs.TransactionHash, SideChainTxs.BlockHeight FROM SideChainTxs WHERE GenesisBlockAddress=? GROUP BY TransactionHash`, genesisBlockAddress)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var txHashes []string
	var blockHeights []uint32
	for rows.Next() {
		var txHash string
		var blockHeight uint32
		err = rows.Scan(&txHash, &blockHeight)
		if err != nil {
			return nil, nil, err
		}
		txHashes = append(txHashes, txHash)
		blockHeights = append(blockHeights, blockHeight)
	}
	return txHashes, blockHeights, nil
}

func (store *DataStoreSideChainImpl) GetSideChainTxsFromHashes(transactionHashes []string) ([]*Transaction, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	var txs []*Transaction
	for _, txHash := range transactionHashes {
		rows, err := store.Query(`SELECT SideChainTxs.TransactionData FROM SideChainTxs WHERE TransactionHash=?`, txHash)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var transactionBytes []byte
			err = rows.Scan(&transactionBytes)
			if err != nil {
				return nil, err
			}

			var tx Transaction
			reader := bytes.NewReader(transactionBytes)
			tx.Deserialize(reader)

			txs = append(txs, &tx)
		}
		rows.Close()
	}

	return txs, nil
}

func (store *DataStoreSideChainImpl) GetSideChainTxsFromHashesAndGenesisAddress(transactionHashes []string, genesisBlockAddress string) ([]*Transaction, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	var txs []*Transaction
	for _, txHash := range transactionHashes {
		rows, err := store.Query(`SELECT SideChainTxs.TransactionData FROM SideChainTxs WHERE TransactionHash=? AND GenesisBlockAddress=? GROUP BY TransactionHash, GenesisBlockAddress`, txHash, genesisBlockAddress)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var transactionBytes []byte
			err = rows.Scan(&transactionBytes)
			if err != nil {
				return nil, err
			}

			var tx Transaction
			reader := bytes.NewReader(transactionBytes)
			tx.Deserialize(reader)

			txs = append(txs, &tx)
		}
		rows.Close()
	}

	return txs, nil
}

func (store *DataStoreMainChainImpl) ResetDataStore() error {
	store.DB.Close()
	os.Remove(DBNameMainChain)

	var err error
	store.DB, err = initMainChainDB()
	if err != nil {
		return err
	}

	return nil
}

func (store *DataStoreMainChainImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.mux.Lock()
		store.DB.Close()
		os.Exit(-1)
	})
}

func (store *DataStoreMainChainImpl) AddMainChainTx(transactionHash, genesisBlockAddress string, transaction *Transaction, proof *bloom.MerkleProof) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO MainChainTxs(TransactionHash, GenesisBlockAddress, TransactionData, MerkleProof) values(?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Serialize transaction
	buf := new(bytes.Buffer)
	transaction.Serialize(buf)
	transactionBytes := buf.Bytes()

	// Serialize merkleProof
	buf = new(bytes.Buffer)
	proof.Serialize(buf)
	merkleProofBytes := buf.Bytes()

	// Do insert
	_, err = stmt.Exec(transactionHash, genesisBlockAddress, transactionBytes, merkleProofBytes)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreMainChainImpl) HasMainChainTx(transactionHash string) (bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash FROM MainChainTxs WHERE TransactionHash=?`, transactionHash)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (store *DataStoreMainChainImpl) RemoveMainChainTx(transactionHash, genesisBlockAddress string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	stmt, err := store.Prepare("DELETE FROM MainChainTxs WHERE TransactionHash=? AND GenesisBlockAddress=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(transactionHash, genesisBlockAddress)
	if err != nil {
		return err
	}

	return nil
}

func (store *DataStoreMainChainImpl) RemoveMainChainTxs(transactionHashes, genesisBlockAddress []string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("DELETE FROM MainChainTxs WHERE TransactionHash=? AND GenesisBlockAddress=?")
	if err != nil {
		return err
	}

	for i := 0; i < len(transactionHashes); i++ {
		_, err = stmt.Exec(transactionHashes[i], genesisBlockAddress[i])
		if err != nil {
			return err
		}
	}
	stmt.Close()
	tx.Commit()

	return nil
}

func (store *DataStoreMainChainImpl) GetAllMainChainTxHashes() ([]string, []string, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash, GenesisBlockAddress FROM MainChainTxs GROUP BY TransactionHash, GenesisBlockAddress`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var txHashes []string
	var genesisAddresses []string
	for rows.Next() {
		var txHash string
		var genesisAddress string
		err = rows.Scan(&txHash, &genesisAddress)
		if err != nil {
			return nil, nil, err
		}
		txHashes = append(txHashes, txHash)
		genesisAddresses = append(genesisAddresses, genesisAddress)
	}
	return txHashes, genesisAddresses, nil
}

func (store *DataStoreMainChainImpl) GetAllMainChainTxs() ([]string, []string, []*Transaction, []*bloom.MerkleProof, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash, GenesisBlockAddress, TransactionData, MerkleProof FROM MainChainTxs GROUP BY TransactionHash, GenesisBlockAddress`)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer rows.Close()

	var txHashes []string
	var genesisAddresses []string
	var txs []*Transaction
	var mps []*bloom.MerkleProof
	for rows.Next() {
		var txHash string
		var genesisAddress string
		var transactionBytes []byte
		var merkleProofBytes []byte
		err = rows.Scan(&txHash, &genesisAddress, &transactionBytes, &merkleProofBytes)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		var tx Transaction
		reader := bytes.NewReader(transactionBytes)
		tx.Deserialize(reader)

		var mp bloom.MerkleProof
		reader = bytes.NewReader(merkleProofBytes)
		mp.Deserialize(reader)

		txHashes = append(txHashes, txHash)
		genesisAddresses = append(genesisAddresses, genesisAddress)
		txs = append(txs, &tx)
		mps = append(mps, &mp)
	}
	return txHashes, genesisAddresses, txs, mps, nil
}

func (store *DataStoreMainChainImpl) GetMainChainTxsFromHashes(transactionHashes []string, genesisBlockAddresses string) ([]*Transaction, []*bloom.MerkleProof, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	var txs []*Transaction
	var mps []*bloom.MerkleProof

	for i := 0; i < len(transactionHashes); i++ {
		rows, err := store.Query(`SELECT MainChainTxs.TransactionData, MainChainTxs.MerkleProof FROM MainChainTxs WHERE TransactionHash=? AND GenesisBlockAddress=? GROUP BY TransactionHash, GenesisBlockAddress`, transactionHashes[i], genesisBlockAddresses)
		if err != nil {
			return nil, nil, err
		}

		for rows.Next() {
			var transactionBytes []byte
			var merkleProofBytes []byte
			err = rows.Scan(&transactionBytes, &merkleProofBytes)
			if err != nil {
				return nil, nil, err
			}

			var tx Transaction
			reader := bytes.NewReader(transactionBytes)
			tx.Deserialize(reader)

			var mp bloom.MerkleProof
			reader = bytes.NewReader(merkleProofBytes)
			mp.Deserialize(reader)

			txs = append(txs, &tx)
			mps = append(mps, &mp)
		}
		rows.Close()
	}

	return txs, mps, nil
}

type DbMainChainFunc struct {
}

func (dbFunc *DbMainChainFunc) GetAvailableUtxos(withdrawBank string) ([]*AddressUTXO, error) {
	utxos, err := DbCache.UTXOStore.GetAddressUTXOsFromGenesisBlockAddress(withdrawBank)
	if err != nil {
		return nil, errors.New("Get spender's UTXOs failed.")
	}
	var availableUTXOs []*AddressUTXO
	var currentHeight = DbCache.UTXOStore.CurrentHeight(QueryHeightCode)
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

func CheckAndCreateDocument(path string) error {
	exist, err := PathExists(path)
	if err != nil {
		return err
	}

	if !exist {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
