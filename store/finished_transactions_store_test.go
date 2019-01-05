package store

import (
	"bytes"
	"testing"

	"github.com/elastos/Elastos.ELA/core"
)

func TestFinishedTxsDataStoreImpl_AddSucceedDepositTxs(t *testing.T) {
	datastore, err := OpenFinishedTxsDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash := "testHash"
	genesisBlockAddress1 := "testAddress1"
	genesisBlockAddress2 := "testAddress2"

	err = datastore.AddSucceedDepositTxs(
		[]string{txHash, txHash},
		[]string{genesisBlockAddress1, genesisBlockAddress2})
	if err != nil {
		t.Error("Add deposit transaction error.")
	}

	ok, err := datastore.HasDepositTx(txHash, genesisBlockAddress1)
	if err != nil {
		t.Error("Check deposit transaction error.")
	}
	if !ok {
		t.Error("Check deposit transaction error.")
	}

	ok, err = datastore.HasDepositTx(txHash, genesisBlockAddress2)
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
	if genesisAddresses[0] != genesisBlockAddress1 || genesisAddresses[1] != genesisBlockAddress2 {
		t.Error("Get deposit transaction error.")
	}

	datastore.ResetDataStore()
}

func TestFinishedTxsDataStoreImpl_AddSucceedDepositTx(t *testing.T) {
	datastore, err := OpenFinishedTxsDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash := "testHash"
	genesisBlockAddress1 := "testAddress1"
	genesisBlockAddress2 := "testAddress2"

	err = datastore.AddSucceedDepositTxs(
		[]string{txHash, txHash},
		[]string{genesisBlockAddress1, genesisBlockAddress2})
	if err != nil {
		t.Error("Add deposit transaction error.")
	}

	ok, err := datastore.HasDepositTx(txHash, genesisBlockAddress1)
	if err != nil {
		t.Error("Check deposit transaction error.")
	}
	if !ok {
		t.Error("Check deposit transaction error.")
	}
	ok, err = datastore.HasDepositTx(txHash, genesisBlockAddress2)
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
	if genesisAddresses[0] != genesisBlockAddress1 || genesisAddresses[1] != genesisBlockAddress2 {
		t.Error("Get deposit transaction error.")
	}

	datastore.ResetDataStore()
}

func TestFinishedTxsDataStoreImpl_GetDepositTxs(t *testing.T) {
	datastore, err := OpenFinishedTxsDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash1 := "testHash1"
	txHash2 := "testHash2"
	genesisBlockAddress1 := "testAddress1"
	genesisBlockAddress2 := "testAddress2"

	err = datastore.AddFailedDepositTxs(
		[]string{txHash1, txHash1},
		[]string{genesisBlockAddress1, genesisBlockAddress2})
	if err != nil {
		t.Error("Add deposit transaction error.")
	}

	err = datastore.AddSucceedDepositTxs(
		[]string{txHash2},
		[]string{genesisBlockAddress2})
	if err != nil {
		t.Error("Add deposit transaction error.")
	}

	failedTxs, genesisBlockAddresses, err := datastore.GetDepositTxs(false)
	if err != nil || len(failedTxs) != 2 || len(genesisBlockAddresses) != 2 {
		t.Error("Get deposit transactions failed.")
	}

	succeedTxs, genesisBlockAddresses, err := datastore.GetDepositTxs(true)
	if err != nil || len(succeedTxs) != 1 || len(genesisBlockAddresses) != 1 {
		t.Error("Get deposit transactions failed.")
	}

	datastore.ResetDataStore()
}

func TestFinishedTxsDataStoreImpl_AddWithdrawTxs(t *testing.T) {
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

	err = datastore.AddFailedWithdrawTxs([]string{txHash1, txHash2}, buf1.Bytes())
	if err != nil {
		t.Error("Add withdraw transaction error.")
	}

	err = datastore.AddSucceedWithdrawTxs([]string{txHash3})
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

	err = datastore.AddSucceedWithdrawTxs([]string{txHash1, txHash2})
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

func TestFinishedTxsDataStoreImpl_GetWithdrawTxs(t *testing.T) {
	datastore, err := OpenFinishedTxsDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash1 := "testHash1"
	txHash2 := "testHash2"
	txHash3 := "testHash3"
	tx1 := core.Transaction{TxType: 0}
	buf1 := new(bytes.Buffer)
	tx1.Serialize(buf1)

	tx2 := core.Transaction{TxType: 1}
	buf2 := new(bytes.Buffer)
	tx2.Serialize(buf2)

	err = datastore.AddFailedWithdrawTxs([]string{txHash1, txHash2}, buf1.Bytes())
	if err != nil {
		t.Error("Add withdraw transaction error.")
	}

	err = datastore.AddSucceedWithdrawTxs([]string{txHash3})
	if err != nil {
		t.Error("Add withdraw transaction error.")
	}

	succeedTxs, err := datastore.GetWithdrawTxs(false)
	if err != nil || len(succeedTxs) != 2 {
		t.Error("Get withdraw transactions error.")
	}

	failedTxs, err := datastore.GetWithdrawTxs(true)
	if err != nil || len(failedTxs) != 1 {
		t.Error("Get withdraw transactions error.")
	}

	datastore.ResetDataStore()
}
