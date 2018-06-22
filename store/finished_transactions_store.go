package store

import (
	"database/sql"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/log"

	_ "github.com/mattn/go-sqlite3"
)

const (
	FinishedTxsDBName = "./DBCache/finishedTxs.db"
)

const (
	//TransactionHash: tx3
	//GenesisBlockAddress: sidechain
	//TransactionData: tx4
	CreateDepositTransactionsTable = `CREATE TABLE IF NOT EXISTS DepositTransactions (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR,
				GenesisBlockAddress VARCHAR(34),
				TransactionData BLOB,
				Succeed BOOLEAN,
				RecordTime TEXT
			);`
	CreateWithdrawTransactionsTable = `CREATE TABLE IF NOT EXISTS WithdrawTransactions (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR,
				SideChainTransactionId INTEGER,
				Succeed BOOLEAN,
				RecordTime TEXT
			);`
	CreateSideChainTransactionsTable = `CREATE TABLE IF NOT EXISTS SideChainTransactions (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionData BLOB,
				RecordTime TEXT
			);`
)

var (
	FinishedTxsDbCache FinishedTransactionsDataStore
)

type FinishedTransactionsDataStore interface {
	AddDepositTx(transactionHash, genesisBlockAddress string, transactionInfoBytes []byte, succeed bool) error
	AddSucceedDepositTx(transactionHash, genesisBlockAddress string) error
	HasDepositTx(transactionHash string) (bool, error)
	GetDepositTxByHash(transactionHash string) ([]bool, []string, error)
	GetDepositTxByHashAndGenesisAddress(transactionHash string, genesisAddress string) (bool, error)

	AddWithdrawTx(transactionHashes []string, transactionByte []byte, succeed bool) error
	AddSucceedWIthdrawTx(transactionHashes []string) error
	HasWithdrawTx(transactionHash string) (bool, error)
	GetWithdrawTxByHash(transactionHash string) (bool, []byte, error)

	AddSideChainTx(transactionByte []byte) error
	GetSideChainTx(sideChainTransactionId uint64) ([]byte, error)

	ResetDataStore() error
}

type FinishedTxsDataStoreImpl struct {
	mux *sync.Mutex

	*sql.DB
}

func OpenFinishedTxsDataStore() (FinishedTransactionsDataStore, error) {
	db, err := initFinishedTxsDB()
	if err != nil {
		return nil, err
	}
	dataStore := &FinishedTxsDataStoreImpl{DB: db, mux: new(sync.Mutex)}

	// Handle system interrupt signals
	dataStore.catchSystemSignals()

	return dataStore, nil
}

func initFinishedTxsDB() (*sql.DB, error) {
	err := CheckAndCreateDocument(DBDocumentNAME)
	if err != nil {
		log.Error("Create DBCache doucument error:", err)
		return nil, err
	}
	db, err := sql.Open(DriverName, FinishedTxsDBName)
	if err != nil {
		log.Error("Open data db error:", err)
		return nil, err
	}
	// Create error deposit transactions table
	_, err = db.Exec(CreateDepositTransactionsTable)
	if err != nil {
		return nil, err
	}
	// Create error withdraw transactions table
	_, err = db.Exec(CreateWithdrawTransactionsTable)
	if err != nil {
		return nil, err
	}
	// Create error side chain transactions table
	_, err = db.Exec(CreateSideChainTransactionsTable)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (store *FinishedTxsDataStoreImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.mux.Lock()
		store.Close()
		os.Exit(-1)
	})
}

func (store *FinishedTxsDataStoreImpl) ResetDataStore() error {

	store.DB.Close()
	os.Remove(FinishedTxsDBName)

	var err error
	store.DB, err = initFinishedTxsDB()
	if err != nil {
		return err
	}

	return nil
}

func (store *FinishedTxsDataStoreImpl) AddDepositTx(transactionHash, genesisBlockAddress string, transactionInfoBytes []byte, succeed bool) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO DepositTransactions(TransactionHash, GenesisBlockAddress, TransactionData, Succeed, RecordTime) values(?,?,?,?,?)")
	if err != nil {
		return err
	}

	// Do insert
	_, err = stmt.Exec(transactionHash, genesisBlockAddress, transactionInfoBytes, succeed, time.Now().Format("2006-01-02_15.04.05"))
	if err != nil {
		return err
	}

	return nil
}

func (store *FinishedTxsDataStoreImpl) AddSucceedDepositTx(transactionHash, genesisBlockAddress string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO DepositTransactions(TransactionHash, GenesisBlockAddress, Succeed, RecordTime) values(?,?,?,?)")
	if err != nil {
		return err
	}

	// Do insert
	_, err = stmt.Exec(transactionHash, genesisBlockAddress, true, time.Now().Format("2006-01-02_15.04.05"))
	if err != nil {
		return err
	}

	return nil
}

func (store *FinishedTxsDataStoreImpl) HasDepositTx(transactionHash string) (bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT GenesisBlockAddress FROM DepositTransactions WHERE TransactionHash=?`, transactionHash)
	defer rows.Close()
	if err != nil {
		return false, err
	}

	return rows.Next(), nil
}

func (store *FinishedTxsDataStoreImpl) GetDepositTxByHash(transactionHash string) ([]bool, []string, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT Succeed, GenesisBlockAddress FROM DepositTransactions WHERE TransactionHash=?`, transactionHash)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var succeed []bool
	var addresses []string
	for rows.Next() {
		var genesisBlockAddress string
		var suc bool
		err = rows.Scan(&suc, &genesisBlockAddress)
		if err != nil {
			return nil, nil, err
		}

		succeed = append(succeed, suc)
		addresses = append(addresses, genesisBlockAddress)
	}
	return succeed, addresses, nil
}

func (store *FinishedTxsDataStoreImpl) GetDepositTxByHashAndGenesisAddress(transactionHash string, genesisAddress string) (bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT Succeed FROM DepositTransactions WHERE TransactionHash=? AND GenesisBlockAddress=? GROUP BY TransactionHash, GenesisBlockAddress`, transactionHash, genesisAddress)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var suc bool
	if rows.Next() {
		err = rows.Scan(&suc)
		if err != nil {
			return false, err
		}
	}

	return suc, nil
}

func (store *FinishedTxsDataStoreImpl) AddWithdrawTx(transactionHashes []string, transactionByte []byte, succeed bool) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO SideChainTransactions(TransactionData, RecordTime) values(?,?)")
	if err != nil {
		return err
	}

	// Do insert
	_, err = stmt.Exec(transactionByte, time.Now().Format("2006-01-02_15.04.05"))
	if err != nil {
		return err
	}
	stmt.Close()

	// Get id
	rows, err := store.Query(`SELECT MAX(Id) FROM SideChainTransactions`)
	if err != nil {
		return err
	}

	if !rows.Next() {
		return errors.New("get max id from SideChainTransactions table failed")
	}
	var sideChainTransactionId int
	err = rows.Scan(&sideChainTransactionId)
	if err != nil {
		return err
	}
	rows.Close()

	tx, err := store.Begin()
	if err != nil {
		return err
	}

	// Prepare sql statement
	stmt, err = tx.Prepare("INSERT INTO WithdrawTransactions(TransactionHash, SideChainTransactionId, Succeed, RecordTime) values(?,?,?,?)")
	if err != nil {
		return err
	}

	// Do insert
	for _, txHash := range transactionHashes {
		_, err = stmt.Exec(txHash, sideChainTransactionId, succeed, time.Now().Format("2006-01-02_15.04.05"))
		if err != nil {
			return err
		}
	}
	stmt.Close()
	tx.Commit()
	return nil
}

func (store *FinishedTxsDataStoreImpl) AddSucceedWIthdrawTx(transactionHashes []string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}

	// Prepare sql statement
	stmt, err := tx.Prepare("INSERT INTO WithdrawTransactions(TransactionHash, SideChainTransactionId, Succeed, RecordTime) values(?,?,?,?)")
	if err != nil {
		return err
	}

	// Do insert
	for _, txHash := range transactionHashes {
		_, err = stmt.Exec(txHash, 0, true, time.Now().Format("2006-01-02_15.04.05"))
		if err != nil {
			return err
		}
	}
	stmt.Close()
	tx.Commit()
	return nil
}

func (store *FinishedTxsDataStoreImpl) HasWithdrawTx(transactionHash string) (bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT Succeed FROM WithdrawTransactions WHERE TransactionHash=?`, transactionHash)
	defer rows.Close()
	if err != nil {
		return false, err
	}

	return rows.Next(), nil
}

func (store *FinishedTxsDataStoreImpl) GetWithdrawTxByHash(transactionHash string) (bool, []byte, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT SideChainTransactionId, Succeed FROM WithdrawTransactions WHERE TransactionHash=? LIMIT 1`, transactionHash)
	if err != nil {
		return false, nil, err
	}
	if !rows.Next() {
		return false, nil, errors.New("get withdraw transaction by hash failed")
	}
	var sideChainTransactionId int
	var succeed bool
	err = rows.Scan(&sideChainTransactionId, &succeed)
	if err != nil {
		return false, nil, err
	}
	rows.Close()

	if succeed {
		return true, nil, err
	}

	rows, err = store.Query(`SELECT TransactionData FROM SideChainTransactions WHERE Id=?`, sideChainTransactionId)
	if err != nil {
		return false, nil, err
	}

	if !rows.Next() {
		return false, nil, errors.New("get withdraw transaction by hash failed, SideChainTransactions table has no record of needed id")
	}
	var transactionBytes []byte
	err = rows.Scan(&transactionBytes)
	if err != nil {
		return false, nil, err
	}

	return false, transactionBytes, nil
}

func (store *FinishedTxsDataStoreImpl) AddSideChainTx(transactionByte []byte) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO SideChainTransactions(TransactionData, RecordTime) values(?,?)")
	if err != nil {
		return err
	}

	// Do insert
	_, err = stmt.Exec(transactionByte, time.Now().Format("2006-01-02_15.04.05"))
	if err != nil {
		return err
	}

	return nil
}

func (store *FinishedTxsDataStoreImpl) GetSideChainTx(sideChainTransactionId uint64) ([]byte, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionData FROM SideChainTransactions WHERE Id=?`, sideChainTransactionId)
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
