package store

import (
	"os"
	"testing"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA/bloom"
	. "github.com/elastos/Elastos.ELA/core"
)

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func setup() {
	config.InitMockConfig()
}

func TestDataStoreImpl_AddSideChainTx(t *testing.T) {
	datastore, err := OpenDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress := "testAddress"
	txHash := "testHash"

	ok, err := datastore.HasSideChainTx(txHash)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	tx := &Transaction{Payload: new(PayloadWithdrawAsset)}
	if err := datastore.AddSideChainTx(txHash, genesisBlockAddress, tx); err != nil {
		t.Error("Add side chain transaction error.")
	}

	ok, err = datastore.HasSideChainTx(txHash)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_RemoveSideChainTxs(t *testing.T) {
	datastore, err := OpenDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress := "testAddress"
	txHash := "testHash"
	tx := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}

	genesisBlockAddress2 := "testAddress2"
	txHash2 := "testHash2"
	tx2 := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}

	datastore.AddSideChainTx(txHash, genesisBlockAddress, tx)
	datastore.AddSideChainTx(txHash2, genesisBlockAddress2, tx2)

	if ok, err := datastore.HasSideChainTx(txHash); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}
	if ok, err := datastore.HasSideChainTx(txHash2); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}

	var removedHashes []string
	removedHashes = append(removedHashes, txHash)
	datastore.RemoveSideChainTxs(removedHashes)

	ok, err := datastore.HasSideChainTx(txHash)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	if ok, err := datastore.HasSideChainTx(txHash2); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_GetAllSideChainTxHashes(t *testing.T) {
	datastore, err := OpenDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress := "testAddress"
	txHash := "testHash"
	txHash2 := "testHash2"

	genesisBlockAddress2 := "testAddress2"
	txHash3 := "testHash3"

	tx := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}
	datastore.AddSideChainTx(txHash, genesisBlockAddress, tx)
	datastore.AddSideChainTx(txHash2, genesisBlockAddress, tx)
	datastore.AddSideChainTx(txHash3, genesisBlockAddress2, tx)

	txHashes, err := datastore.GetAllSideChainTxHashes(genesisBlockAddress)
	if err != nil {
		t.Error("Get all side chain transactions error.")
	}
	if len(txHashes) != 2 {
		t.Error("Get all side chain transactions error.")
	}
	for _, hash := range txHashes {
		if hash != txHash && hash != txHash2 {
			t.Error("Get all side chain transactions error.")
		}
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_GetSideChainTxsFromHashes(t *testing.T) {
	datastore, err := OpenDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress := "testAddress"
	txHash := "testHash"
	txHash2 := "testHash2"

	genesisBlockAddress2 := "testAddress2"
	txHash3 := "testHash3"

	tx1 := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}
	tx2 := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}
	tx3 := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}

	tx1.LockTime = 1
	tx2.LockTime = 2
	tx3.LockTime = 3

	datastore.AddSideChainTx(txHash, genesisBlockAddress, tx1)
	datastore.AddSideChainTx(txHash2, genesisBlockAddress, tx2)
	datastore.AddSideChainTx(txHash3, genesisBlockAddress2, tx3)

	var txHashes []string
	txHashes = append(txHashes, txHash)
	txHashes = append(txHashes, txHash2)
	txHashes = append(txHashes, txHash3)

	txs, err := datastore.GetSideChainTxsFromHashes(txHashes)
	if err != nil {
		t.Error("Get all side chain transactions error.")
	}
	if len(txs) != 3 {
		t.Error("Get all side chain transactions error.")
	}
	for _, tx := range txs {
		if tx.LockTime != 1 && tx.LockTime != 2 && tx.LockTime != 3 {
			t.Error("Get all side chain transactions error.")
		}
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_AddMainChainTx(t *testing.T) {
	datastore, err := OpenDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash := "testHash"

	ok, err := datastore.HashMainChainTx(txHash)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	tx := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}
	mp := new(bloom.MerkleProof)
	if err := datastore.AddMainChainTx(txHash, tx, mp); err != nil {
		t.Error("Add main chain transaction error.")
	}

	ok, err = datastore.HashMainChainTx(txHash)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_RemoveMainChainTxs(t *testing.T) {
	datastore, err := OpenDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash := "testHash"
	txHash2 := "testHash2"

	tx := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}
	tx2 := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}

	mp := new(bloom.MerkleProof)
	mp2 := new(bloom.MerkleProof)

	datastore.AddMainChainTx(txHash, tx, mp)
	datastore.AddMainChainTx(txHash2, tx2, mp2)

	if ok, err := datastore.HashMainChainTx(txHash); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}
	if ok, err := datastore.HashMainChainTx(txHash2); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}

	var removedHashes []string
	removedHashes = append(removedHashes, txHash)
	datastore.RemoveMainChainTxs(removedHashes)

	ok, err := datastore.HashMainChainTx(txHash)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	if ok, err := datastore.HashMainChainTx(txHash2); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_GetAllMainChainTxHashes(t *testing.T) {
	datastore, err := OpenDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash := "testHash"
	txHash2 := "testHash2"
	txHash3 := "testHash3"

	tx := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}
	tx2 := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}
	tx3 := &Transaction{TxType: WithdrawAsset, Payload: new(PayloadWithdrawAsset)}

	mp := new(bloom.MerkleProof)
	mp2 := new(bloom.MerkleProof)
	mp3 := new(bloom.MerkleProof)

	datastore.AddMainChainTx(txHash, tx, mp)
	datastore.AddMainChainTx(txHash2, tx2, mp2)
	datastore.AddMainChainTx(txHash3, tx3, mp3)

	txHashes, err := datastore.GetAllMainChainTxHashes()
	if err != nil {
		t.Error("Get all main chain transactions error.")
	}
	if len(txHashes) != 3 {
		t.Error("Get all main chain transactions error.")
	}
	for _, hash := range txHashes {
		if hash != txHash && hash != txHash2 && hash != txHash3 {
			t.Error("Get all main chain transactions error.")
		}
	}

	proofs, _, err := datastore.GetMainChainTxsFromHashes([]string{txHash, txHash2, txHash3})
	if err != nil {
		t.Error("Get main chain txs from hashes error.")
	}
	if len(proofs) != 3 {
		t.Error("Get main chain txs from hashes error.")
	}

	datastore.ResetDataStore()
}
