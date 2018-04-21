package wallet

import (
	"bytes"
	"database/sql"
	"math"
	"os"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/log"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA.Utility/core"

	_ "github.com/mattn/go-sqlite3"
)

/*
钱包的数据仓库，存储UTXO，合约脚本等，使用SQLite
*/
const (
	DriverName      = "sqlite3"
	DBName          = "./wallet.db"
	QueryHeightCode = 0
	ResetHeightCode = math.MaxUint32
)

const (
	CreateInfoTable = `CREATE TABLE IF NOT EXISTS Info (
				Name VARCHAR(20) NOT NULL PRIMARY KEY,
				Value BLOB
			);`
	CreateAddressesTable = `CREATE TABLE IF NOT EXISTS Addresses (
				Id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
				ProgramHash BLOB UNIQUE NOT NULL,
				RedeemScript BLOB UNIQUE NOT NULL
			);`
	CreateUTXOsTable = `CREATE TABLE IF NOT EXISTS UTXOs (
				OutPoint BLOB NOT NULL PRIMARY KEY,
				Amount BLOB NOT NULL,
				LockTime INTEGER NOT NULL,
				AddressId INTEGER NOT NULL,
				FOREIGN KEY(AddressId) REFERENCES Addresses(Id)
			);`
)

type Address struct {
	Address      string
	ProgramHash  *Uint168
	RedeemScript []byte
}

type AddressUTXO struct {
	Op       *OutPoint
	Amount   *Fixed64
	LockTime uint32
}

type DataStore interface {
	sync.Locker
	DataSync

	CurrentHeight(height uint32) uint32

	AddAddress(programHash *Uint168, redeemScript []byte) error
	DeleteAddress(programHash *Uint168) error
	GetAddressInfo(programHash *Uint168) (*Address, error)
	GetAddresses() ([]*Address, error)

	AddAddressUTXO(programHash *Uint168, utxo *AddressUTXO) error
	DeleteUTXO(input *OutPoint) error
	GetAddressUTXOs(programHash *Uint168) ([]*AddressUTXO, error)

	ResetDataStore() error
}

type DataStoreImpl struct {
	sync.Mutex
	DataSync

	*sql.DB
}

func OpenDataStore() (DataStore, error) {
	db, err := initDB()
	if err != nil {
		return nil, err
	}
	dataStore := &DataStoreImpl{DB: db}

	dataStore.DataSync = GetDataSync(dataStore)

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
	// Create addresses table
	_, err = db.Exec(CreateAddressesTable)
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

func (store *DataStoreImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.Lock()
		store.Close()
	})
}

func (store *DataStoreImpl) ResetDataStore() error {

	addresses, err := store.GetAddresses()
	if err != nil {
		return err
	}

	store.DB.Close()
	os.Remove(DBName)

	store.DB, err = initDB()
	if err != nil {
		return err
	}

	for _, address := range addresses {
		err = store.AddAddress(address.ProgramHash, address.RedeemScript)
		if err != nil {
			return err
		}
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

func (store *DataStoreImpl) AddAddress(programHash *Uint168, redeemScript []byte) error {
	store.Lock()
	defer store.Unlock()

	stmt, err := store.Prepare("INSERT INTO Addresses(ProgramHash, RedeemScript) values(?,?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(programHash.Bytes(), redeemScript)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) DeleteAddress(programHash *Uint168) error {
	store.Lock()
	defer store.Unlock()

	// Find addressId by ProgramHash
	row := store.QueryRow("SELECT Id FROM Addresses WHERE ProgramHash=?", programHash.Bytes())
	var addressId int
	err := row.Scan(&addressId)
	if err != nil {
		return err
	}

	// Delete UTXOs of this address
	stmt, err := store.Prepare(
		"DELETE FROM UTXOs WHERE AddressId=?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(addressId)
	if err != nil {
		return err
	}

	// Delete address from address table
	stmt, err = store.Prepare("DELETE FROM Addresses WHERE Id=?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(addressId)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) GetAddressInfo(programHash *Uint168) (*Address, error) {
	store.Lock()
	defer store.Unlock()

	// Query address info by it's ProgramHash
	sql := `SELECT RedeemScript FROM Addresses WHERE ProgramHash=?`
	row := store.QueryRow(sql, programHash.Bytes())
	var redeemScript []byte
	err := row.Scan(&redeemScript)
	if err != nil {
		return nil, err
	}
	address, err := programHash.ToAddress()
	if err != nil {
		return nil, err
	}
	return &Address{address, programHash, redeemScript}, nil
}

func (store *DataStoreImpl) GetAddresses() ([]*Address, error) {
	store.Lock()
	defer store.Unlock()

	rows, err := store.Query("SELECT ProgramHash, RedeemScript FROM Addresses")
	if err != nil {
		log.Error("Get address query error:", err)
		return nil, err
	}
	defer rows.Close()

	var addresses []*Address
	for rows.Next() {
		var programHashBytes []byte
		var redeemScript []byte
		err = rows.Scan(&programHashBytes, &redeemScript)
		if err != nil {
			log.Error("Get address scan row:", err)
			return nil, err
		}
		programHash, err := Uint168FromBytes(programHashBytes)
		if err != nil {
			return nil, err
		}
		address, err := programHash.ToAddress()
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, &Address{address, programHash, redeemScript})
	}
	return addresses, nil
}

func (store *DataStoreImpl) AddAddressUTXO(programHash *Uint168, utxo *AddressUTXO) error {
	store.Lock()
	defer store.Unlock()

	// Find addressId by ProgramHash
	row := store.QueryRow("SELECT Id FROM Addresses WHERE ProgramHash=?", programHash.Bytes())
	var addressId int
	err := row.Scan(&addressId)
	if err != nil {
		return err
	}
	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO UTXOs(OutPoint, Amount, LockTime, AddressId) values(?,?,?,?)")
	if err != nil {
		return err
	}
	// Serialize input
	buf := new(bytes.Buffer)
	utxo.Op.Serialize(buf)
	opBytes := buf.Bytes()
	// Serialize amount
	buf = new(bytes.Buffer)
	utxo.Amount.Serialize(buf)
	amountBytes := buf.Bytes()
	// Do insert
	_, err = stmt.Exec(opBytes, amountBytes, utxo.LockTime, addressId)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) DeleteUTXO(op *OutPoint) error {
	store.Lock()
	defer store.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("DELETE FROM UTXOs WHERE OutPoint=?")
	if err != nil {
		return err
	}
	// Serialize input
	buf := new(bytes.Buffer)
	op.Serialize(buf)
	opBytes := buf.Bytes()
	// Do delete
	_, err = stmt.Exec(opBytes)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) GetAddressUTXOs(programHash *Uint168) ([]*AddressUTXO, error) {
	store.Lock()
	defer store.Unlock()

	rows, err := store.Query(`SELECT UTXOs.OutPoint, UTXOs.Amount, UTXOs.LockTime FROM UTXOs INNER JOIN Addresses
 								ON UTXOs.AddressId=Addresses.Id WHERE Addresses.ProgramHash=?`, programHash.Bytes())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inputs []*AddressUTXO
	for rows.Next() {
		var opBytes []byte
		var amountBytes []byte
		var lockTime uint32
		err = rows.Scan(&opBytes, &amountBytes, &lockTime)
		if err != nil {
			return nil, err
		}

		var op OutPoint
		reader := bytes.NewReader(opBytes)
		op.Deserialize(reader)

		var amount Fixed64
		reader = bytes.NewReader(amountBytes)
		amount.Deserialize(reader)

		inputs = append(inputs, &AddressUTXO{&op, &amount, lockTime})
	}
	return inputs, nil
}
