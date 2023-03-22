package store

import (
	"bytes"
	"database/sql"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA.SPV/bloom"
	"github.com/elastos/Elastos.ELA/common"
	elatx "github.com/elastos/Elastos.ELA/core/transaction"
	elacommon "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	_ "github.com/mattn/go-sqlite3"
)

var (
	DBDocumentNAME          = filepath.Join(config.DataPath, config.DataDir, "arbiter")
	DBNameMainChain         = filepath.Join(DBDocumentNAME, "mainChainCache.db")
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
				Name VARCHAR(20) NOT NULL PRIMARY KEY,
				Value BLOB
			);`
	CreateSideChainTxsTable = `CREATE TABLE IF NOT EXISTS SideChainTxs (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR UNIQUE,
				TransactionData BLOB,
				BlockHeight INTEGER
			);`
	CreateNFTDestroyTxsTable = `CREATE TABLE IF NOT EXISTS NFTDestroyTxs (
				Id INTEGER NOT NULL PRIMARY KEY,
				NFTID VARCHAR UNIQUE,
				TransactionData BLOB,
				BlockHeight INTEGER
			);`
	CreateReturnDepositTransactionsTable = `CREATE TABLE IF NOT EXISTS ReturnDepositTransactions (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR,
				GenesisBlockAddress VARCHAR(34),
				TransactionData BLOB,
				RecordTime TEXT
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
	Input               *elacommon.Input
	Amount              *common.Fixed64
	GenesisBlockAddress string
}

type DataStore interface {
	ResetDataStore(dbName string) error
	catchSystemSignals()
}

type DataStoreMainChain interface {
	DataStore

	CurrentHeight(height uint32) uint32
	BestHeight(id peer.PID) uint64
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

	SideChainName() string
	GenesisBlockAddress() string

	CurrentSideHeight(height uint32) uint32
	AddSideChainTx(tx *base.SideChainTransaction) error
	AddSideChainTxs(txs []*base.SideChainTransaction) error
	HasSideChainTx(transactionHash string) (bool, error)
	RemoveSideChainTxs(transactionHashes []string) error
	GetAllSideChainTxHashes() ([]string, error)
	GetAllSideChainTxHashesAndHeights() ([]string, []uint32, error)
	GetSideChainTxsFromHashes(transactionHashes []string) ([]*base.WithdrawTx, error)

	AddReturnDepositTx(txid string, genesisBlockAddress string, transactionByte []byte) error
	GetReturnDepositTx(txid string) ([]byte, error)
	GetAllReturnDepositTx(genesisBlockAddress string) ([][]byte, []string, error)
	GetAllReturnDepositTxs() ([]string, error)
	RemoveReturnDepositTxs(transactionHashes []string) error

	RemoveNFTDestroyTxs(NFTIDS []string) error
	HasNFTDestroyTx(NFTID string) (bool, error)
	AddNFTDestroyTx(tx *base.NFTDestroyTransaction) error
	AddNFTDestroyTxs(txs []*base.NFTDestroyTransaction) error
	GetAllNFTDestroyID() ([]string, error)
	GetNFTDestroyTxsFromIDs(nftIDs []string) ([]*base.NFTDestroyFromSideChainTx, error)
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
	SideChainStore           []DataStoreSideChain
	RegisteredSideChainStore DataStoreRegisteredSideChain
}

func (d *DataStoreImpl) GetDataStoreByDBName(sideChainName string) DataStoreSideChain {
	for _, s := range d.SideChainStore {
		if sideChainName == s.SideChainName() {
			return s
		}
	}

	return nil
}

func (d *DataStoreImpl) GetDataStoreGenesisBlocAddress(genesisBlockAddress string) DataStoreSideChain {
	for _, s := range d.SideChainStore {
		if genesisBlockAddress == s.GenesisBlockAddress() {
			return s
		}
	}

	return nil
}

type DataStoreMainChainImpl struct {
	mux *sync.Mutex

	*sql.DB

	// height of main chain
	mainChainHeight uint32
}

type DataStoreSideChainImpl struct {
	mux *sync.Mutex

	*sql.DB

	sideChainName       string
	genesisBlockAddress string
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

	scStore := make([]DataStoreSideChain, 0)
	for i, s := range dbSideChain {
		scStore = append(scStore, &DataStoreSideChainImpl{
			mux:                 new(sync.Mutex),
			DB:                  s,
			sideChainName:       config.Parameters.SideNodeList[i].Name,
			genesisBlockAddress: config.Parameters.SideNodeList[i].GenesisBlockAddress,
		})
	}
	dataStore := &DataStoreImpl{
		MainChainStore:           &DataStoreMainChainImpl{mux: new(sync.Mutex), DB: dbMainChain},
		RegisteredSideChainStore: &DataStoreRegisteredSideChainStoreImpl{mux: new(sync.Mutex), DB: registerSc},
		SideChainStore:           scStore,
	}

	// Handle system interrupt signals
	dataStore.MainChainStore.catchSystemSignals()
	dataStore.RegisteredSideChainStore.catchSystemSignals()
	for _, s := range dataStore.SideChainStore {
		s.catchSystemSignals()
	}

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

func OpenSideChainDataStore() ([]*DataStoreSideChainImpl, error) {
	dbSideChain, err := initSideChainDB()
	if err != nil {
		return nil, err
	}
	scStore := make([]*DataStoreSideChainImpl, 0)
	for i, s := range dbSideChain {
		scStore = append(scStore, &DataStoreSideChainImpl{mux: new(sync.Mutex),
			DB: s, sideChainName: config.Parameters.SideNodeList[i].Name})
	}

	for _, s := range scStore {
		// Handle system interrupt signals
		s.catchSystemSignals()
	}

	return scStore, nil
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

func CreateSideChainDBByConfig(sideChain *config.SideNodeConfig) (*DataStoreSideChainImpl, error) {

	DBNameSideChain := filepath.Join(DBDocumentNAME, sideChain.Name+"_sideChainCache.db")
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
	// Create return deposit transactions table
	_, err = db.Exec(CreateReturnDepositTransactionsTable)
	if err != nil {
		return nil, err
	}
	// Create return nft destroy transactions table
	_, err = db.Exec(CreateNFTDestroyTxsTable)
	if err != nil {
		return nil, err
	}

	stmt, err := db.Prepare("INSERT INTO SideHeightInfo(Name, Value) values(?,?)")
	if err != nil {
		return nil, err
	}
	_, err = stmt.Exec("Height", uint32(0))
	if err != nil {
		return nil, err
	}

	return &DataStoreSideChainImpl{
		mux:                 new(sync.Mutex),
		DB:                  db,
		sideChainName:       sideChain.Name,
		genesisBlockAddress: sideChain.GenesisBlockAddress,
	}, nil
}

func initSideChainDB() ([]*sql.DB, error) {
	err := CheckAndCreateDocument(DBDocumentNAME)
	if err != nil {
		log.Error("Create DBCache doucument error:", err)
		return nil, err
	}

	result := make([]*sql.DB, 0)
	for _, sideChain := range config.Parameters.SideNodeList {

		DBNameSideChain := filepath.Join(DBDocumentNAME, sideChain.Name+"_sideChainCache.db")
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
		// Create return deposit transactions table
		_, err = db.Exec(CreateReturnDepositTransactionsTable)
		if err != nil {
			return nil, err
		}
		// Create return nft destroy transactions table
		_, err = db.Exec(CreateNFTDestroyTxsTable)
		if err != nil {
			return nil, err
		}

		stmt, err := db.Prepare("INSERT INTO SideHeightInfo(Name, Value) values(?,?)")
		if err != nil {
			return nil, err
		}
		stmt.Exec("Height", uint32(0))
		result = append(result, db)
	}

	return result, nil
}

func initSideChainDBByName(sideChainName string) (*sql.DB, error) {
	err := CheckAndCreateDocument(DBDocumentNAME)
	if err != nil {
		log.Error("Create DBCache doucument error:", err)
		return nil, err
	}

	for _, sideChain := range config.Parameters.SideNodeList {
		if sideChain.Name != sideChainName {
			continue
		}

		DBNameSideChain := filepath.Join(DBDocumentNAME, sideChainName+"_sideChainCache.db")
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
		// Create return deposit transactions table
		_, err = db.Exec(CreateReturnDepositTransactionsTable)
		if err != nil {
			return nil, err
		}
		// Create return nft destroy transactions table
		_, err = db.Exec(CreateNFTDestroyTxsTable)
		if err != nil {
			return nil, err
		}
		stmt, err := db.Prepare("INSERT INTO SideHeightInfo(Name, Value) values(?,?)")
		if err != nil {
			return nil, err
		}
		stmt.Exec("Height", uint32(0))
		return db, nil

	}

	return nil, errors.New("not found db")
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

// just for test
func (store *DataStoreSideChainImpl) ResetDataStore(dbName string) error {
	store.DB.Close()

	os.Remove(dbName)

	var err error
	store.DB, err = initSideChainDBByName(dbName)
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

func (store *DataStoreSideChainImpl) SideChainName() string {
	return store.sideChainName
}

func (store *DataStoreSideChainImpl) GenesisBlockAddress() string {
	return store.genesisBlockAddress
}

func (store *DataStoreSideChainImpl) CurrentSideHeight(height uint32) uint32 {
	store.mux.Lock()
	defer store.mux.Unlock()

	row := store.QueryRow("SELECT Value FROM SideHeightInfo WHERE Name=?", "Height")
	var storedHeight uint32
	row.Scan(&storedHeight)
	if height > storedHeight {
		// Received reset height code
		if height == ResetHeightCode {
			height = 0
		}

		// Insert current height
		stmt, err := store.Prepare("UPDATE SideHeightInfo SET Value=? WHERE Name=?")
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

func (store *DataStoreSideChainImpl) AddSideChainTxs(txs []*base.SideChainTransaction) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	// Prepare sql statement
	stmt, err := tx.Prepare("INSERT INTO SideChainTxs(TransactionHash, TransactionData, BlockHeight) values(?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Do insert
	for _, tx := range txs {
		_, err = stmt.Exec(tx.TransactionHash, tx.Transaction, tx.BlockHeight)
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
	stmt, err := store.Prepare("INSERT INTO SideChainTxs(TransactionHash, TransactionData, BlockHeight) values(?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Do insert
	_, err = stmt.Exec(tx.TransactionHash, tx.Transaction, tx.BlockHeight)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreSideChainImpl) HasSideChainTx(transactionHash string) (bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT Id FROM SideChainTxs WHERE TransactionHash=?`, transactionHash)
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

func (store *DataStoreSideChainImpl) GetAllSideChainTxHashesAndHeights() ([]string, []uint32, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT SideChainTxs.TransactionHash, SideChainTxs.BlockHeight FROM SideChainTxs`)
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

func (store *DataStoreSideChainImpl) AddNFTDestroyTx(tx *base.NFTDestroyTransaction) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO NFTDestroyTxs(NFTID, TransactionData, BlockHeight) values(?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Do insert
	_, err = stmt.Exec(tx.ID, tx.Transaction, tx.BlockHeight)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreSideChainImpl) HasNFTDestroyTx(NFTID string) (bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT NFTID FROM NFTDestroyTxs WHERE NFTID=?`, NFTID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (store *DataStoreSideChainImpl) RemoveNFTDestroyTxs(NFTIDS []string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	stmt, err := tx.Prepare("DELETE FROM NFTDestroyTxs WHERE NFTID=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, nftID := range NFTIDS {
		stmt.Exec(nftID)
	}

	return nil
}

func (store *DataStoreSideChainImpl) AddNFTDestroyTxs(txs []*base.NFTDestroyTransaction) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	// Prepare sql statement
	stmt, err := tx.Prepare("INSERT INTO NFTDestroyTxs(NFTID, TransactionData, BlockHeight) values(?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Do insert
	for _, tx := range txs {
		_, err = stmt.Exec(tx.ID, tx.Transaction, tx.BlockHeight)
		if err != nil {
			log.Error("[AddNFTDestroyTxs] err")
			continue
		}
	}
	return nil
}

func (store *DataStoreSideChainImpl) GetAllNFTDestroyID() ([]string, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT NFTDestroyTxs.NFTID FROM NFTDestroyTxs`)
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

func (store *DataStoreSideChainImpl) GetNFTDestroyTxsFromIDs(nftIDs []string) ([]*base.NFTDestroyFromSideChainTx, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	var txs []*base.NFTDestroyFromSideChainTx
	var buf bytes.Buffer
	buf.WriteString("SELECT NFTDestroyTxs.TransactionData FROM NFTDestroyTxs WHERE NFTID IN (")
	hashesLen := len(nftIDs)
	for index, nftID := range nftIDs {
		buf.WriteString("'")
		buf.WriteString(nftID)
		buf.WriteString("'")
		if index == hashesLen-1 {
			buf.WriteString(")")
		} else {
			buf.WriteString(",")
		}
	}
	buf.WriteString(" GROUP BY NFTID")

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

		tx := new(base.NFTDestroyFromSideChainTx)
		reader := bytes.NewReader(transactionBytes)
		tx.Deserialize(reader)
		txs = append(txs, tx)

	}
	return txs, nil
}

func (store *DataStoreSideChainImpl) AddReturnDepositTx(txid string, genesisBlockAddress string, transactionByte []byte) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO ReturnDepositTransactions(TransactionHash, GenesisBlockAddress, TransactionData, RecordTime) values(?,?,?,?)")
	if err != nil {
		return err
	}

	// Do insert
	_, err = stmt.Exec(txid, genesisBlockAddress, transactionByte, time.Now().Format("2006-01-02_15.04.05"))
	if err != nil {
		return err
	}

	return nil
}

func (store *DataStoreSideChainImpl) GetAllReturnDepositTx(genesisBlockAddress string) ([][]byte, []string, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionData,TransactionHash FROM ReturnDepositTransactions WHERE GenesisBlockAddress=?`, genesisBlockAddress)
	defer rows.Close()
	if err != nil {
		return nil, nil, err
	}

	transactionArrayBytes := make([][]byte, 0)
	transactionArrayHash := make([]string, 0)
	for rows.Next() {
		var transactionBytes []byte
		var transactionHash string
		err = rows.Scan(&transactionBytes, &transactionHash)
		if err != nil {
			return nil, nil, err
		}
		transactionArrayBytes = append(transactionArrayBytes, transactionBytes)
		transactionArrayHash = append(transactionArrayHash, transactionHash)
	}

	return transactionArrayBytes, transactionArrayHash, nil
}

func (store *DataStoreSideChainImpl) GetAllReturnDepositTxs() ([]string, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash FROM ReturnDepositTransactions`)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	transactionArrayHash := make([]string, 0)
	for rows.Next() {
		var transactionHash string
		err = rows.Scan(&transactionHash)
		if err != nil {
			return nil, err
		}
		transactionArrayHash = append(transactionArrayHash, transactionHash)
	}

	return transactionArrayHash, nil
}

func (store *DataStoreSideChainImpl) RemoveReturnDepositTxs(transactionHashes []string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	stmt, err := tx.Prepare("DELETE FROM ReturnDepositTransactions WHERE TransactionHash=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, txHash := range transactionHashes {
		stmt.Exec(txHash)
	}

	return nil
}

func (store *DataStoreSideChainImpl) GetReturnDepositTx(txid string) ([]byte, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionData FROM ReturnDepositTransactions WHERE TransactionHash=?`, txid)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	var transactionBytes []byte
	err = rows.Scan(&transactionBytes)
	if err != nil {
		return nil, err
	}

	return transactionBytes, nil
}

func (store *DataStoreMainChainImpl) ResetDataStore(dbName string) error {
	store.DB.Close()
	os.Remove(dbName)

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

	store.mainChainHeight = storedHeight
	return storedHeight
}

func (store *DataStoreMainChainImpl) BestHeight(id peer.PID) uint64 {
	store.mux.Lock()
	defer store.mux.Unlock()

	if store.mainChainHeight != 0 {
		return uint64(store.mainChainHeight)
	}

	row := store.QueryRow("SELECT Value FROM Info WHERE Name=?", "Height")
	var storedHeight uint32
	row.Scan(&storedHeight)

	store.mainChainHeight = storedHeight

	return uint64(storedHeight)
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

		r := bytes.NewReader(transactionBytes)
		tx, err := elatx.GetTransactionByBytes(r)
		if err != nil {
			return nil, err
		}
		tx.Deserialize(r)

		var mp bloom.MerkleProof
		reader := bytes.NewReader(merkleProofBytes)
		mp.Deserialize(reader)

		txs = append(txs, &base.MainChainTransaction{txHash,
			genesisAddress, tx, &mp})
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

			r := bytes.NewReader(transactionBytes)
			tx, err := elatx.GetTransactionByBytes(r)
			if err != nil {
				return nil, err
			}
			tx.Deserialize(r)

			var mp bloom.MerkleProof
			reader := bytes.NewReader(merkleProofBytes)
			mp.Deserialize(reader)

			spvTxs = append(spvTxs, &base.SpvTransaction{MainChainTransaction: tx, Proof: &mp})
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

func (store *DataStoreRegisteredSideChainStoreImpl) ResetDataStore(dbName string) error {
	store.DB.Close()
	os.Remove(dbName)

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
	stmt, err := tx.Prepare("INSERT INTO RegisteredSideChains(TransactionHash, GenesisBlockAddress, RegisterInfo) values(?,?,?)")
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
 									RegisterInfo FROM RegisteredSideChains where TransactionHash = ?`, tx)
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
 									RegisterInfo FROM RegisteredSideChains`)
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

	sql := `SELECT RegisterInfo FROM RegisteredSideChains WHERE TransactionHash=? AND GenesisBlockAddress=?`
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
