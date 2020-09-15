package store

import (
	"bytes"
	"database/sql"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA.SPV/bloom"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	_ "github.com/mattn/go-sqlite3"
)

var (
	DBDocumentNAME          = filepath.Join(config.DataPath, config.DataDir, "arbiter")
	DBNameMainChain         = filepath.Join(DBDocumentNAME, "mainChainCache.db")
	DBNameSideChain         = filepath.Join(DBDocumentNAME, "sideChainCache.db")
	DBNameRegisterSideChain = filepath.Join(DBDocumentNAME, "registerSideChainCache.db")
)

const (
	DriverName = "sqlite3"

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
	CreateSideChainTxsTable = `CREATE TABLE IF NOT EXISTS SideChainTxs (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR UNIQUE,
				GenesisBlockAddress VARCHAR(34),
				TransactionData BLOB,
				BlockHeight INTEGER
			);`
	CreateMainChainTxsTable = `CREATE TABLE IF NOT EXISTS MainChainTxs (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR,
				GenesisBlockAddress VARCHAR(34),
				TransactionData BLOB,
				MerkleProof BLOB,
                UNIQUE (TransactionHash, GenesisBlockAddress)
			);`

	CreateRegisteredSideChainsTable = `CREATE TABLE IF NOT EXISTS RegisteredSideChains (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR,
				GenesisBlockAddress VARCHAR(34),
				RegisterInfo BLOB,
				UNIQUE (TransactionHash, GenesisBlockAddress)
			);`
)

var (
	DbCache DataStoreImpl
)

type AddressUTXO struct {
	Input               *types.Input
	Amount              *common.Fixed64
	GenesisBlockAddress string
}

type DataStore interface {
	ResetDataStore() error
	catchSystemSignals()
}

type DataStoreMainChain interface {
	DataStore

	CurrentHeight(height uint32) uint32
	AddMainChainTx(tx *base.MainChainTransaction) error
	AddMainChainTxs(txs []*base.MainChainTransaction) ([]bool, error)
	HasMainChainTx(transactionHash, genesisBlockAddress string) (bool, error)
	RemoveMainChainTx(transactionHash, genesisBlockAddress string) error
	RemoveMainChainTxs(transactionHashes, genesisBlockAddress []string) error
	GetAllMainChainTxHashes() ([]string, []string, error)
	GetAllMainChainTxs() ([]*base.MainChainTransaction, error)
	GetMainChainTxsFromHashes(transactionHashes []string, genesisBlockAddresses string) ([]*base.SpvTransaction, error)
}

type DataStoreSideChain interface {
	DataStore

	CurrentSideHeight(genesisBlockAddress string, height uint32) uint32
	AddSideChainTx(tx *base.SideChainTransaction) error
	AddSideChainTxs(txs []*base.SideChainTransaction) error
	HasSideChainTx(transactionHash string) (bool, error)
	RemoveSideChainTxs(transactionHashes []string) error
	GetAllSideChainTxHashes() ([]string, error)
	GetAllSideChainTxHashesAndHeights(genesisBlockAddress string) ([]string, []uint32, error)
	GetSideChainTxsFromHashes(transactionHashes []string) ([]*base.WithdrawTx, error)
	GetSideChainTxsFromHashesAndGenesisAddress(transactionHashes []string, genesisBlockAddress string) ([]*base.WithdrawTx, error)
}

type DataStoreRegisteredSideChain interface {
	DataStore

	CurrentHeight(height uint32) uint32
	AddRegisteredSideChainTx(tx *base.RegisteredSideChainTransaction) error
	AddRegisteredSideChainTxs(txs []*base.RegisteredSideChainTransaction) ([]bool, error)
	HasRegisteredSideChainTx(transactionHash, genesisBlockAddress string) (bool, error)
	RemoveRegisteredSideChainTx(transactionHash, genesisBlockAddress string) error
	RemoveRegisteredSideChainTxs(transactionHashes, genesisBlockAddress []string) error
	GetAllRegisteredSideChainTxsHashes() ([]string, []string, error)
	GetAllRegisteredSideChainTxs() ([]*base.RegisteredSideChainTransaction, error)
	GetRegisteredSideChainTxByHash(tx string) (*base.RegisteredSideChainTransaction, error)
	GetRegisteredSideChainTxsFromHashes(transactionHashes []string, genesisBlockAddresses string) ([]*base.RegisteredSideChain, error)
}

type DataStoreImpl struct {
	MainChainStore           DataStoreMainChain
	SideChainStore           DataStoreSideChain
	RegisteredSideChainStore DataStoreRegisteredSideChain
}

type DataStoreMainChainImpl struct {
	mux *sync.Mutex

	*sql.DB
}

type DataStoreSideChainImpl struct {
	mux *sync.Mutex

	*sql.DB
}

type DataStoreRegisteredSideChainStoreImpl struct {
	mux *sync.Mutex

	*sql.DB
}

func OpenDataStore() (*DataStoreImpl, error) {
	if err := checkAndCreateArbiterDataDir(); err != nil {
		log.Errorf("create arbiter db dir error: %s\n", err)
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
	registerSc, err := initRegisterSideChainDB()
	if err != nil {
		return nil, err
	}
	dataStore := &DataStoreImpl{
		MainChainStore:           &DataStoreMainChainImpl{mux: new(sync.Mutex), DB: dbMainChain},
		SideChainStore:           &DataStoreSideChainImpl{mux: new(sync.Mutex), DB: dbSideChain},
		RegisteredSideChainStore: &DataStoreRegisteredSideChainStoreImpl{mux: new(sync.Mutex), DB: registerSc},
	}

	// Handle system interrupt signals
	dataStore.MainChainStore.catchSystemSignals()
	dataStore.SideChainStore.catchSystemSignals()
	dataStore.RegisteredSideChainStore.catchSystemSignals()

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

func OpenRegisteredSideChainDataStore() (*DataStoreRegisteredSideChainStoreImpl, error) {
	dbRegisterSideChain, err := initRegisterSideChainDB()
	if err != nil {
		return nil, err
	}
	dataStore := &DataStoreRegisteredSideChainStoreImpl{mux: new(sync.Mutex), DB: dbRegisterSideChain}

	// Handle system interrupt signals
	dataStore.catchSystemSignals()

	return dataStore, nil
}

func checkAndCreateArbiterDataDir() error {
	arbiterPath := filepath.Join(config.DataPath, config.DataDir, config.ArbiterDir)
	if _, err := os.Stat(arbiterPath); os.IsNotExist(err) {
		if err := os.MkdirAll(arbiterPath, 0740); err != nil {
			return err
		}
	}
	return nil
}

func initMainChainDB() (*sql.DB, error) {
	err := CheckAndCreateDocument(DBDocumentNAME)
	if err != nil {
		log.Error("create DBCache doucument error:", err)
		return nil, err
	}
	db, err := sql.Open(DriverName, DBNameMainChain)
	if err != nil {
		log.Error("Open data db error:", err)
		return nil, err
	}
	// Create info table
	_, err = db.Exec(CreateInfoTable)
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

func initRegisterSideChainDB() (*sql.DB, error) {
	err := CheckAndCreateDocument(DBDocumentNAME)
	if err != nil {
		log.Error("Create DBCache doucument error:", err)
		return nil, err
	}
	db, err := sql.Open(DriverName, DBNameRegisterSideChain)
	if err != nil {
		log.Error("Open data db error:", err)
		return nil, err
	}

	// Create SideChainTxs table
	_, err = db.Exec(CreateRegisteredSideChainsTable)
	if err != nil {
		return nil, err
	}

	return db, nil
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

func (store *DataStoreSideChainImpl) AddSideChainTxs(txs []*base.SideChainTransaction) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	// Prepare sql statement
	stmt, err := tx.Prepare("INSERT INTO SideChainTxs(TransactionHash, GenesisBlockAddress, TransactionData, BlockHeight) values(?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Do insert
	for _, tx := range txs {
		_, err = stmt.Exec(tx.TransactionHash, tx.GenesisBlockAddress, tx.Transaction, tx.BlockHeight)
		if err != nil {
			log.Error("[AddSideChainTxs] err")
			continue
		}
	}

	return nil
}

func (store *DataStoreSideChainImpl) AddSideChainTx(tx *base.SideChainTransaction) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO SideChainTxs(TransactionHash, GenesisBlockAddress, TransactionData, BlockHeight) values(?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Do insert
	_, err = stmt.Exec(tx.TransactionHash, tx.GenesisBlockAddress, tx.Transaction, tx.BlockHeight)
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
	defer tx.Commit()

	stmt, err := tx.Prepare("DELETE FROM SideChainTxs WHERE TransactionHash=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, txHash := range transactionHashes {
		stmt.Exec(txHash)
	}

	return nil
}

func (store *DataStoreSideChainImpl) GetAllSideChainTxHashes() ([]string, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT SideChainTxs.TransactionHash FROM SideChainTxs`)
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

	rows, err := store.Query(`SELECT SideChainTxs.TransactionHash, SideChainTxs.BlockHeight FROM SideChainTxs WHERE GenesisBlockAddress=?`, genesisBlockAddress)
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

func (store *DataStoreSideChainImpl) GetSideChainTxsFromHashes(transactionHashes []string) ([]*base.WithdrawTx, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	var txs []*base.WithdrawTx
	var buf bytes.Buffer
	buf.WriteString("SELECT SideChainTxs.TransactionData FROM SideChainTxs WHERE TransactionHash IN (")
	hashesLen := len(transactionHashes)
	for index, hash := range transactionHashes {
		buf.WriteString("'")
		buf.WriteString(hash)
		buf.WriteString("'")
		if index == hashesLen-1 {
			buf.WriteString(")")
		} else {
			buf.WriteString(",")
		}
	}
	buf.WriteString(" GROUP BY TransactionHash")

	rows, err := store.Query(buf.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var transactionBytes []byte
		err = rows.Scan(&transactionBytes)
		if err != nil {
			return nil, err
		}

		tx := new(base.WithdrawTx)
		reader := bytes.NewReader(transactionBytes)
		tx.Deserialize(reader)
		txs = append(txs, tx)

	}
	return txs, nil
}

func (store *DataStoreSideChainImpl) GetSideChainTxsFromHashesAndGenesisAddress(transactionHashes []string, genesisBlockAddress string) ([]*base.WithdrawTx, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	var txs []*base.WithdrawTx
	for _, txHash := range transactionHashes {
		rows, err := store.Query(`SELECT SideChainTxs.TransactionData FROM SideChainTxs WHERE TransactionHash=? AND GenesisBlockAddress=?`, txHash, genesisBlockAddress)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var transactionBytes []byte
			err = rows.Scan(&transactionBytes)
			if err != nil {
				rows.Close()
				return nil, err
			}

			tx := new(base.WithdrawTx)
			reader := bytes.NewReader(transactionBytes)
			tx.Deserialize(reader)

			txs = append(txs, tx)
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

func (store *DataStoreMainChainImpl) CurrentHeight(height uint32) uint32 {
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

func (store *DataStoreMainChainImpl) AddMainChainTx(tx *base.MainChainTransaction) error {
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
	tx.Transaction.Serialize(buf)
	transactionBytes := buf.Bytes()

	// Serialize merkleProof
	buf = new(bytes.Buffer)
	tx.Proof.Serialize(buf)
	merkleProofBytes := buf.Bytes()

	// Do insert
	_, err = stmt.Exec(tx.TransactionHash, tx.GenesisBlockAddress, transactionBytes, merkleProofBytes)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreMainChainImpl) AddMainChainTxs(txs []*base.MainChainTransaction) ([]bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Commit()

	// Prepare sql statement
	stmt, err := tx.Prepare("INSERT INTO MainChainTxs(TransactionHash, GenesisBlockAddress, TransactionData, MerkleProof) values(?,?,?,?)")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var result []bool
	for _, tx := range txs {
		// Serialize transaction
		buf := new(bytes.Buffer)
		tx.Transaction.Serialize(buf)
		transactionBytes := buf.Bytes()

		// Serialize merkleProof
		buf = new(bytes.Buffer)
		tx.Proof.Serialize(buf)
		merkleProofBytes := buf.Bytes()

		// Do insert
		_, err = stmt.Exec(tx.TransactionHash, tx.GenesisBlockAddress, transactionBytes, merkleProofBytes)
		if err != nil {
			result = append(result, false)
		} else {
			result = append(result, true)
		}
	}

	return result, nil
}

func (store *DataStoreMainChainImpl) HasMainChainTx(transactionHash, genesisBlockAddress string) (bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	sql := `SELECT TransactionHash FROM MainChainTxs WHERE TransactionHash=? AND GenesisBlockAddress=?`
	rows, err := store.Query(sql, transactionHash, genesisBlockAddress)
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
	defer tx.Commit()

	stmt, err := tx.Prepare("DELETE FROM MainChainTxs WHERE TransactionHash=? AND GenesisBlockAddress=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := 0; i < len(transactionHashes); i++ {
		_, err = stmt.Exec(transactionHashes[i], genesisBlockAddress[i])
		if err != nil {
			continue
		}
	}

	return nil
}

func (store *DataStoreMainChainImpl) GetAllMainChainTxHashes() ([]string, []string, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash, GenesisBlockAddress FROM MainChainTxs`)
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

func (store *DataStoreMainChainImpl) GetAllMainChainTxs() ([]*base.MainChainTransaction, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash, GenesisBlockAddress,
 									TransactionData, MerkleProof FROM MainChainTxs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*base.MainChainTransaction
	for rows.Next() {
		var txHash string
		var genesisAddress string
		var transactionBytes []byte
		var merkleProofBytes []byte
		err = rows.Scan(&txHash, &genesisAddress, &transactionBytes, &merkleProofBytes)
		if err != nil {
			return nil, err
		}

		var tx types.Transaction
		reader := bytes.NewReader(transactionBytes)
		tx.Deserialize(reader)

		var mp bloom.MerkleProof
		reader = bytes.NewReader(merkleProofBytes)
		mp.Deserialize(reader)

		txs = append(txs, &base.MainChainTransaction{txHash,
			genesisAddress, &tx, &mp})
	}
	return txs, nil
}

func (store *DataStoreMainChainImpl) GetMainChainTxsFromHashes(transactionHashes []string,
	genesisBlockAddresses string) ([]*base.SpvTransaction, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	var spvTxs []*base.SpvTransaction

	sql := `SELECT TransactionData, MerkleProof FROM MainChainTxs WHERE TransactionHash=? AND GenesisBlockAddress=?`
	for i := 0; i < len(transactionHashes); i++ {
		rows, err := store.Query(sql, transactionHashes[i], genesisBlockAddresses)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var transactionBytes []byte
			var merkleProofBytes []byte
			err = rows.Scan(&transactionBytes, &merkleProofBytes)
			if err != nil {
				rows.Close()
				return nil, err
			}

			var tx types.Transaction
			reader := bytes.NewReader(transactionBytes)
			tx.Deserialize(reader)

			var mp bloom.MerkleProof
			reader = bytes.NewReader(merkleProofBytes)
			mp.Deserialize(reader)

			spvTxs = append(spvTxs, &base.SpvTransaction{MainChainTransaction: &tx, Proof: &mp})
		}
		rows.Close()
	}

	return spvTxs, nil
}

func CheckAndCreateDocument(path string) error {
	exist, err := PathExists(path)
	if err != nil {
		return err
	}
	if !exist {
		err := os.MkdirAll(path, 0700)
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

func (store *DataStoreRegisteredSideChainStoreImpl) ResetDataStore() error {
	store.DB.Close()
	os.Remove(DBNameMainChain)

	var err error
	store.DB, err = initMainChainDB()
	if err != nil {
		return err
	}

	return nil
}

func (store *DataStoreRegisteredSideChainStoreImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.mux.Lock()
		store.DB.Close()
		os.Exit(-1)
	})
}

func (store *DataStoreRegisteredSideChainStoreImpl) CurrentHeight(height uint32) uint32 {
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

func (store *DataStoreRegisteredSideChainStoreImpl) AddRegisteredSideChainTx(tx *base.RegisteredSideChainTransaction) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO RegisteredSideChains(TransactionHash, GenesisBlockAddress, RegisterInfo) values(?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Serialize transaction
	buf := new(bytes.Buffer)
	tx.RegisteredSideChain.Serialize(buf)
	transactionBytes := buf.Bytes()

	// Do insert
	_, err = stmt.Exec(tx.TransactionHash, tx.GenesisBlockAddress, transactionBytes)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreRegisteredSideChainStoreImpl) AddRegisteredSideChainTxs(txs []*base.RegisteredSideChainTransaction) ([]bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Commit()

	// Prepare sql statement
	stmt, err := tx.Prepare("INSERT INTO RegisteredSideChains(TransactionHash, GenesisBlockAddress, TransactionData, MerkleProof) values(?,?,?,?)")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var result []bool
	for _, tx := range txs {
		// Serialize transaction
		buf := new(bytes.Buffer)
		tx.RegisteredSideChain.Serialize(buf)
		transactionBytes := buf.Bytes()

		// Do insert
		_, err = stmt.Exec(tx.TransactionHash, tx.GenesisBlockAddress, transactionBytes)
		if err != nil {
			result = append(result, false)
		} else {
			result = append(result, true)
		}
	}

	return result, nil
}

func (store *DataStoreRegisteredSideChainStoreImpl) HasRegisteredSideChainTx(transactionHash, genesisBlockAddress string) (bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	sql := `SELECT TransactionHash FROM RegisteredSideChains WHERE TransactionHash=? AND GenesisBlockAddress=?`
	rows, err := store.Query(sql, transactionHash, genesisBlockAddress)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (store *DataStoreRegisteredSideChainStoreImpl) RemoveRegisteredSideChainTx(transactionHash, genesisBlockAddress string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	stmt, err := store.Prepare("DELETE FROM RegisteredSideChains WHERE TransactionHash=? AND GenesisBlockAddress=?")
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

func (store *DataStoreRegisteredSideChainStoreImpl) RemoveRegisteredSideChainTxs(transactionHashes, genesisBlockAddress []string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	stmt, err := tx.Prepare("DELETE FROM RegisteredSideChains WHERE TransactionHash=? AND GenesisBlockAddress=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := 0; i < len(transactionHashes); i++ {
		_, err = stmt.Exec(transactionHashes[i], genesisBlockAddress[i])
		if err != nil {
			continue
		}
	}

	return nil
}

func (store *DataStoreRegisteredSideChainStoreImpl) GetAllRegisteredSideChainTxsHashes() ([]string, []string, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash, GenesisBlockAddress FROM RegisteredSideChains`)
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

func (store *DataStoreRegisteredSideChainStoreImpl) GetRegisteredSideChainTxByHash(tx string) (*base.RegisteredSideChainTransaction, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash, GenesisBlockAddress,
 									TransactionData FROM RegisteredSideChains where TransactionHash = ?`, tx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs *base.RegisteredSideChainTransaction
	for rows.Next() {
		var txHash string
		var genesisAddress string
		var transactionBytes []byte
		err = rows.Scan(&txHash, &genesisAddress, &transactionBytes)
		if err != nil {
			return nil, err
		}

		var tx base.RegisteredSideChain
		reader := bytes.NewReader(transactionBytes)
		tx.Deserialize(reader)

		txs = &base.RegisteredSideChainTransaction{txHash,
			genesisAddress, &tx}
		break
	}
	return txs, nil
}

func (store *DataStoreRegisteredSideChainStoreImpl) GetAllRegisteredSideChainTxs() ([]*base.RegisteredSideChainTransaction, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash, GenesisBlockAddress,
 									TransactionData FROM RegisteredSideChains`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*base.RegisteredSideChainTransaction
	for rows.Next() {
		var txHash string
		var genesisAddress string
		var transactionBytes []byte
		err = rows.Scan(&txHash, &genesisAddress, &transactionBytes)
		if err != nil {
			return nil, err
		}

		var tx base.RegisteredSideChain
		reader := bytes.NewReader(transactionBytes)
		tx.Deserialize(reader)

		txs = append(txs, &base.RegisteredSideChainTransaction{txHash,
			genesisAddress, &tx})
	}
	return txs, nil
}

func (store *DataStoreRegisteredSideChainStoreImpl) GetRegisteredSideChainTxsFromHashes(transactionHashes []string,
	genesisBlockAddresses string) ([]*base.RegisteredSideChain, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	var rsc []*base.RegisteredSideChain

	sql := `SELECT TransactionData FROM RegisteredSideChains WHERE TransactionHash=? AND GenesisBlockAddress=?`
	for i := 0; i < len(transactionHashes); i++ {
		rows, err := store.Query(sql, transactionHashes[i], genesisBlockAddresses)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var transactionBytes []byte
			err = rows.Scan(&transactionBytes)
			if err != nil {
				rows.Close()
				return nil, err
			}

			var tx base.RegisteredSideChain
			reader := bytes.NewReader(transactionBytes)
			tx.Deserialize(reader)

			rsc = append(rsc, &tx)
		}
		rows.Close()
	}

	return rsc, nil
}
