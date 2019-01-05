package store

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/log"

	_ "github.com/mattn/go-sqlite3"
)

var FinishedTxsDBName = filepath.Join(DBDocumentNAME, "finishedTxs.db")

const (
	//TransactionHash: tx3
	//GenesisBlockAddress: sidechain
	//TransactionData: tx4
	CreateDepositTransactionsTable = `CREATE TABLE IF NOT EXISTS DepositTransactions (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR,
				GenesisBlockAddress VARCHAR(34),
				Succeed BOOLEAN,
				RecordTime TEXT,
				UNIQUE (TransactionHash, GenesisBlockAddress)
			);`
	CreateWithdrawTransactionsTable = `CREATE TABLE IF NOT EXISTS WithdrawTransactions (
				Id INTEGER NOT NULL PRIMARY KEY,
				TransactionHash VARCHAR UNIQUE,
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
	AddFailedDepositTxs(transactionHashes, genesisBlockAddresses []string) error
	AddSucceedDepositTxs(transactionHashes, genesisBlockAddresses []string) error
	HasDepositTx(transactionHash string, genesisBlockAddress string) (bool, error)
	GetDepositTxByHash(transactionHash string) ([]bool, []string, error)
	GetDepositTxByHashAndGenesisAddress(transactionHash string, genesisAddress string) (bool, error)
	GetDepositTxs(succeed bool) ([]string, []string, error)

	AddFailedWithdrawTxs(transactionHashes []string, transactionByte []byte) error
	AddSucceedWithdrawTxs(transactionHashes []string) error
	HasWithdrawTx(transactionHash string) (bool, error)
	GetWithdrawTxByHash(transactionHash string) (bool, []byte, error)
	GetWithdrawTxs(succeed bool) ([]string, error)

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

func (store *FinishedTxsDataStoreImpl) AddFailedDepositTxs(transactionHashes, genesisBlockAddresses []string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	// Prepare sql statement
	stmt, err := tx.Prepare("INSERT INTO DepositTransactions(TransactionHash, GenesisBlockAddress, Succeed, RecordTime) values(?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Do insert
	for i := 0; i < len(transactionHashes); i++ {
		_, err = stmt.Exec(transactionHashes[i], genesisBlockAddresses[i], false, time.Now().Format("2006-01-02_15.04.05"))
		if err != nil {
			continue
		}
	}
	return nil
}

func (store *FinishedTxsDataStoreImpl) AddSucceedDepositTxs(transactionHashes, genesisBlockAddresses []string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	// Prepare sql statement
	stmt, err := tx.Prepare("INSERT INTO DepositTransactions(TransactionHash, GenesisBlockAddress, Succeed, RecordTime) values(?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Do insert
	for i := 0; i < len(transactionHashes); i++ {
		_, err = stmt.Exec(transactionHashes[i], genesisBlockAddresses[i], true, time.Now().Format("2006-01-02_15.04.05"))
		if err != nil {
			continue
		}
	}
	return nil
}

func (store *FinishedTxsDataStoreImpl) HasDepositTx(transactionHash string, genesisBlockAddress string) (bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT GenesisBlockAddress FROM DepositTransactions WHERE TransactionHash=? AND GenesisBlockAddress=?`, transactionHash, genesisBlockAddress)
	if err != nil {
		return false, err
	}
	defer rows.Close()

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
	var genesisAddresses []string
	for rows.Next() {
		var address string
		var suc bool
		err = rows.Scan(&suc, &address)
		if err != nil {
			return nil, nil, err
		}

		succeed = append(succeed, suc)
		genesisAddresses = append(genesisAddresses, address)
	}
	return succeed, genesisAddresses, nil
}

func (store *FinishedTxsDataStoreImpl) GetDepositTxByHashAndGenesisAddress(transactionHash string, genesisAddress string) (bool, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT Succeed FROM DepositTransactions WHERE TransactionHash=? AND GenesisBlockAddress=?`, transactionHash, genesisAddress)
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

func (store *FinishedTxsDataStoreImpl) GetDepositTxs(succeed bool) ([]string, []string, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash, GenesisBlockAddress FROM DepositTransactions WHERE Succeed=?`, succeed)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var txHashes []string
	var genesisAddresses []string
	for rows.Next() {
		var hash string
		var address string
		err = rows.Scan(&hash, &address)
		if err != nil {
			return nil, nil, err
		}

		txHashes = append(txHashes, hash)
		genesisAddresses = append(genesisAddresses, address)
	}

	return txHashes, genesisAddresses, nil
}

func (store *FinishedTxsDataStoreImpl) AddFailedWithdrawTxs(transactionHashes []string, transactionByte []byte) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	// Do insert
	_, err := store.Exec("INSERT INTO SideChainTransactions(TransactionData, RecordTime) values(?,?)",
		transactionByte, time.Now().Format("2006-01-02_15.04.05"))
	if err != nil {
		return err
	}

	// Get id
	var sideChainTransactionId int
	row := store.QueryRow(`SELECT MAX(Id) FROM SideChainTransactions`)
	err = row.Scan(&sideChainTransactionId)
	if err != nil {
		return err
	}

	tx, err := store.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	// Prepare sql statement
	stmt2, err := tx.Prepare("INSERT INTO WithdrawTransactions(TransactionHash, SideChainTransactionId, Succeed, RecordTime) values(?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt2.Close()

	// Do insert
	for _, txHash := range transactionHashes {
		_, err = stmt2.Exec(txHash, sideChainTransactionId, false, time.Now().Format("2006-01-02_15.04.05"))
		if err != nil {
			continue
		}
	}
	return nil
}

func (store *FinishedTxsDataStoreImpl) AddSucceedWithdrawTxs(transactionHashes []string) error {
	store.mux.Lock()
	defer store.mux.Unlock()

	tx, err := store.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	// Prepare sql statement
	stmt, err := tx.Prepare("INSERT INTO WithdrawTransactions(TransactionHash, SideChainTransactionId, Succeed, RecordTime) values(?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Do insert
	for _, txHash := range transactionHashes {
		if _, err := stmt.Exec(txHash, 0, true, time.Now().Format("2006-01-02_15.04.05")); err != nil {
			log.Error("[AddSucceedWithdrawTxs] txHash:", txHash, "err:", err.Error())
		}
	}
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
	defer rows.Close()
	if !rows.Next() {
		return false, nil, errors.New("get withdraw transaction by hash failed")
	}
	var sideChainTransactionId int
	var succeed bool
	err = rows.Scan(&sideChainTransactionId, &succeed)
	if err != nil {
		return false, nil, err
	}

	if succeed {
		return true, nil, err
	}

	rowsS, err := store.Query(`SELECT TransactionData FROM SideChainTransactions WHERE Id=?`, sideChainTransactionId)
	if err != nil {
		return false, nil, err
	}
	defer rowsS.Close()

	if !rowsS.Next() {
		return false, nil, errors.New("get withdraw transaction by hash failed, SideChainTransactions table has no record of needed id")
	}
	var transactionBytes []byte
	err = rowsS.Scan(&transactionBytes)
	if err != nil {
		return false, nil, err
	}

	return false, transactionBytes, nil
}

func (store *FinishedTxsDataStoreImpl) GetWithdrawTxs(succeed bool) ([]string, error) {
	store.mux.Lock()
	defer store.mux.Unlock()

	rows, err := store.Query(`SELECT TransactionHash FROM WithdrawTransactions WHERE Succeed=?`, succeed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txHashes []string
	for rows.Next() {
		var hash string
		err = rows.Scan(&hash)
		if err != nil {
			return nil, err
		}

		txHashes = append(txHashes, hash)
	}

	return txHashes, nil
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
