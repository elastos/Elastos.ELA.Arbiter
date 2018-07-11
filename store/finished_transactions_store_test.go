package store

import (
	"encoding/json"
	"testing"

	"bytes"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA/core"
)

func TestFinishedTxsDataStoreImpl_AddDepositTx(t *testing.T) {
	datastore, err := OpenFinishedTxsDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress := "testAddress"
	genesisBlockAddress2 := "testAddress2"

	txHash := "testHash"
	info := base.TransactionInfo{}
	depositTxBytes, err := json.Marshal(info)
	if err != nil {
		t.Error("Deposit transactionInfo to bytes failed")
	}

	err = datastore.AddDepositTx(txHash, genesisBlockAddress, depositTxBytes, true)
	if err != nil {
		t.Error("Add deposit transaction error.")
	}

	err = datastore.AddDepositTx(txHash, genesisBlockAddress2, depositTxBytes, false)
	if err != nil {
		t.Error("Add deposit transaction error.")
	}

	ok, err := datastore.HasDepositTx(txHash)
	if err != nil {
		t.Error("Check deposit transaction error.")
	}
	if !ok {
		t.Error("Check deposit transaction error.")
	}

	succeedList, genesisAddresses, err := datastore.GetDepositTxByHash(txHash)
	if err != nil {
		t.Error("Get deposit transaction error.")
	}
	if len(succeedList) != 2 || len(genesisAddresses) != 2 {
		t.Error("Get deposit transaction error.")
	}
	if succeedList[0] != true || succeedList[1] != false {
		t.Error("Get deposit transaction error.")
	}
	if genesisAddresses[0] != genesisBlockAddress || genesisAddresses[1] != genesisBlockAddress2 {
		t.Error("Get deposit transaction error.")
	}

	datastore.ResetDataStore()
}

func TestFinishedTxsDataStoreImpl_AddSucceedDepositTx(t *testing.T) {
	datastore, err := OpenFinishedTxsDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress := "testAddress"
	genesisBlockAddress2 := "testAddress2"
	txHash := "testHash"

	err = datastore.AddSucceedDepositTx(txHash, genesisBlockAddress)
	if err != nil {
		t.Error("Add deposit transaction error.")
	}

	err = datastore.AddSucceedDepositTx(txHash, genesisBlockAddress2)
	if err != nil {
		t.Error("Add deposit transaction error.")
	}

	ok, err := datastore.HasDepositTx(txHash)
	if err != nil {
		t.Error("Check deposit transaction error.")
	}
	if !ok {
		t.Error("Check deposit transaction error.")
	}

	succeedList, genesisAddresses, err := datastore.GetDepositTxByHash(txHash)
	if err != nil {
		t.Error("Get deposit transaction error.")
	}
	if len(succeedList) != 2 || len(genesisAddresses) != 2 {
		t.Error("Get deposit transaction error.")
	}
	if succeedList[0] != true || succeedList[1] != true {
		t.Error("Get deposit transaction error.")
	}
	if genesisAddresses[0] != genesisBlockAddress || genesisAddresses[1] != genesisBlockAddress2 {
		t.Error("Get deposit transaction error.")
	}

	datastore.ResetDataStore()
}

func TestFinishedTxsDataStoreImpl_AddWithdrawTx(t *testing.T) {
	datastore, err := OpenFinishedTxsDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash1 := "testHash1"
	txHash2 := "testHash2"
	txHash3 := "testHash3"
	txHash4 := "testHash4"
	tx1 := core.Transaction{TxType: 0}
	buf1 := new(bytes.Buffer)
	tx1.Serialize(buf1)

	tx2 := core.Transaction{TxType: 1}
	buf2 := new(bytes.Buffer)
	tx2.Serialize(buf2)

	err = datastore.AddWithdrawTx([]string{txHash1, txHash2}, buf1.Bytes(), false)
	if err != nil {
		t.Error("Add withdraw transaction error.")
	}

	err = datastore.AddWithdrawTx([]string{txHash3}, buf2.Bytes(), true)
	if err != nil {
		t.Error("Add withdraw transaction error.")
	}

	ok, err := datastore.HasWithdrawTx(txHash1)
	if err != nil {
		t.Error("Check withdraw transaction error.")
	}
	if !ok {
		t.Error("Check withdraw transaction error.")
	}

	ok, err = datastore.HasWithdrawTx(txHash4)
	if err != nil {
		t.Error("Check withdraw transaction error.")
	}
	if ok {
		t.Error("Check withdraw transaction error.")
	}

	// verify txhash1
	succeed, transactionBytes, err := datastore.GetWithdrawTxByHash(txHash1)
	if err != nil {
		t.Error("Get withdraw transaction error.")
	}

	if succeed != false {
		t.Error("Get withdraw transaction error.")
	}

	tx := new(core.Transaction)
	reader := bytes.NewReader(transactionBytes)
	tx.Deserialize(reader)
	if tx.TxType != 0 {
		t.Error("Get withdraw transaction error.")
	}

	// verify txhash2
	succeed, transactionBytes, err = datastore.GetWithdrawTxByHash(txHash2)
	if err != nil {
		t.Error("Get withdraw transaction error.")
	}

	if succeed != false {
		t.Error("Get withdraw transaction error.")
	}

	tx = new(core.Transaction)
	reader = bytes.NewReader(transactionBytes)
	tx.Deserialize(reader)
	if tx.TxType != 0 {
		t.Error("Get withdraw transaction error.")
	}

	// verify txhash3
	succeed, transactionBytes, err = datastore.GetWithdrawTxByHash(txHash3)
	if err != nil {
		t.Error("Get withdraw transaction error.")
	}

	if succeed != true {
		t.Error("Get withdraw transaction error.")
	}

	if transactionBytes != nil {
		t.Error("Get withdraw transaction error.")
	}

	datastore.ResetDataStore()
}

func TestFinishedTxsDataStoreImpl_AddSucceedWIthdrawTx(t *testing.T) {
	datastore, err := OpenFinishedTxsDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash1 := "testHash1"
	txHash2 := "testHash2"

	err = datastore.AddSucceedWithdrawTx([]string{txHash1, txHash2})
	if err != nil {
		t.Error("Add withdraw transaction error.")
	}

	ok, err := datastore.HasWithdrawTx(txHash1)
	if err != nil {
		t.Error("Check withdraw transaction error.")
	}
	if !ok {
		t.Error("Check withdraw transaction error.")
	}

	ok, err = datastore.HasWithdrawTx(txHash2)
	if err != nil {
		t.Error("Check withdraw transaction error.")
	}
	if !ok {
		t.Error("Check withdraw transaction error.")
	}

	// verify txhash1
	succeed, transactionBytes, err := datastore.GetWithdrawTxByHash(txHash1)
	if err != nil {
		t.Error("Get withdraw transaction error.")
	}

	if succeed != true {
		t.Error("Get withdraw transaction error.")
	}
	if transactionBytes != nil {
		t.Error("Get withdraw transaction error.")
	}

	datastore.ResetDataStore()
}
