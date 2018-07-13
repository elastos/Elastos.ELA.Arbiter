package store

import (
	"bytes"
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
	datastore, err := OpenSideChainDataStore()
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

	tx := &Transaction{Payload: new(PayloadWithdrawFromSideChain)}
	if err := datastore.AddSideChainTx(txHash, genesisBlockAddress, tx, 10); err != nil {
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

func TestDataStoreImpl_AddSideChainTxs(t *testing.T) {
	datastore, err := OpenSideChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress1 := "testAddress1"
	genesisBlockAddress2 := "testAddress2"
	genesisBlockAddress3 := "testAddress3"
	txHash1 := "testHash1"
	txHash2 := "testHash2"
	txHash3 := "testHash3"

	ok, err := datastore.HasSideChainTx(txHash1)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}
	ok, err = datastore.HasSideChainTx(txHash2)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}
	ok, err = datastore.HasSideChainTx(txHash3)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	tx := &Transaction{Payload: new(PayloadWithdrawFromSideChain)}
	buf := new(bytes.Buffer)
	tx.Serialize(buf)
	if err := datastore.AddSideChainTxs(
		[]string{txHash1, txHash2, txHash3},
		[]string{genesisBlockAddress1, genesisBlockAddress2, genesisBlockAddress3},
		[][]byte{buf.Bytes(), buf.Bytes(), buf.Bytes()},
		[]uint32{10, 10, 10}); err != nil {
		t.Error("Add side chain transaction error.")
	}

	ok, err = datastore.HasSideChainTx(txHash1)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}
	ok, err = datastore.HasSideChainTx(txHash2)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}
	ok, err = datastore.HasSideChainTx(txHash3)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_RemoveSideChainTxs(t *testing.T) {
	datastore, err := OpenSideChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress := "testAddress"
	txHash := "testHash"
	tx := &Transaction{TxType: WithdrawFromSideChain, Payload: new(PayloadWithdrawFromSideChain)}

	genesisBlockAddress2 := "testAddress2"
	txHash2 := "testHash2"
	tx2 := &Transaction{TxType: WithdrawFromSideChain, Payload: new(PayloadWithdrawFromSideChain)}

	datastore.AddSideChainTx(txHash, genesisBlockAddress, tx, 10)
	datastore.AddSideChainTx(txHash2, genesisBlockAddress2, tx2, 10)

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
	datastore, err := OpenSideChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress := "testAddress"
	txHash := "testHash"
	txHash2 := "testHash2"

	genesisBlockAddress2 := "testAddress2"
	txHash3 := "testHash3"

	tx := &Transaction{TxType: WithdrawFromSideChain, Payload: new(PayloadWithdrawFromSideChain)}
	datastore.AddSideChainTx(txHash, genesisBlockAddress, tx, 10)
	datastore.AddSideChainTx(txHash2, genesisBlockAddress, tx, 10)
	datastore.AddSideChainTx(txHash3, genesisBlockAddress2, tx, 11)
	datastore.AddSideChainTx(txHash3, genesisBlockAddress2, tx, 11)

	txHashes, err := datastore.GetAllSideChainTxHashes()
	if err != nil {
		t.Error("Get all side chain transactions error.")
	}
	if len(txHashes) != 3 {
		t.Error("Get all side chain transactions error.")
	}

	txHashes, heights, err := datastore.GetAllSideChainTxHashesAndHeights(genesisBlockAddress)
	if err != nil {
		t.Error("Get all side chain transactions error.")
	}
	if len(txHashes) != 2 || len(heights) != 2 {
		t.Error("Get all side chain transactions error.")
	}
	for _, hash := range txHashes {
		if hash != txHash && hash != txHash2 {
			t.Error("Get all side chain transactions error.")
		}
	}
	for _, height := range heights {
		if height != 10 {
			t.Error("Get all side chain transactions error.")
		}
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_GetSideChainTxsFromHashes(t *testing.T) {
	datastore, err := OpenSideChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress := "testAddress"
	txHash := "testHash"
	txHash2 := "testHash2"

	genesisBlockAddress2 := "testAddress2"
	txHash3 := "testHash3"

	tx1 := &Transaction{TxType: WithdrawFromSideChain, Payload: new(PayloadWithdrawFromSideChain)}
	tx2 := &Transaction{TxType: WithdrawFromSideChain, Payload: new(PayloadWithdrawFromSideChain)}
	tx3 := &Transaction{TxType: WithdrawFromSideChain, Payload: new(PayloadWithdrawFromSideChain)}

	tx1.LockTime = 1
	tx2.LockTime = 2
	tx3.LockTime = 3

	datastore.AddSideChainTx(txHash, genesisBlockAddress, tx1, 10)
	datastore.AddSideChainTx(txHash2, genesisBlockAddress, tx2, 10)
	datastore.AddSideChainTx(txHash3, genesisBlockAddress2, tx3, 10)

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
	datastore, err := OpenMainChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash := "testHash"
	genesisAddress := "testAddress"

	ok, err := datastore.HasMainChainTx(txHash, genesisAddress)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	tx := &Transaction{TxType: WithdrawFromSideChain, Payload: new(PayloadWithdrawFromSideChain)}
	mp := new(bloom.MerkleProof)
	if err := datastore.AddMainChainTx(txHash, genesisAddress, tx, mp); err != nil {
		t.Error("Add main chain transaction error.")
	}

	ok, err = datastore.HasMainChainTx(txHash, genesisAddress)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_AddMainChainTxs(t *testing.T) {
	datastore, err := OpenMainChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash1 := "testHash1"
	txHash2 := "testHash2"
	txHash3 := "testHash3"
	genesisAddress1 := "testAddress1"
	genesisAddress2 := "testAddress2"
	genesisAddress3 := "testAddress3"

	ok, err := datastore.HasMainChainTx(txHash1, genesisAddress1)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}
	ok, err = datastore.HasMainChainTx(txHash2, genesisAddress2)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}
	ok, err = datastore.HasMainChainTx(txHash3, genesisAddress3)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	tx := &Transaction{TxType: WithdrawFromSideChain, Payload: new(PayloadWithdrawFromSideChain)}
	mp := new(bloom.MerkleProof)
	results, err := datastore.AddMainChainTxs(
		[]string{txHash1, txHash2, txHash3},
		[]string{genesisAddress1, genesisAddress2, genesisAddress3},
		[]*Transaction{tx, tx, tx},
		[]*bloom.MerkleProof{mp, mp, mp})
	if len(results) != 3 {
		t.Error("Add main chain txs failed")
	}
	for _, result := range results {
		if result == false {
			t.Error("Add main chain txs failed")
		}
	}

	ok, err = datastore.HasMainChainTx(txHash1, genesisAddress1)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}
	ok, err = datastore.HasMainChainTx(txHash2, genesisAddress2)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}
	ok, err = datastore.HasMainChainTx(txHash3, genesisAddress3)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_RemoveMainChainTxs(t *testing.T) {
	datastore, err := OpenMainChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash1 := "testHash1"
	txHash2 := "testHash2"
	txHash3 := "testHash3"
	genesisAddress := "genesis"

	tx := &Transaction{TxType: WithdrawFromSideChain, Payload: new(PayloadWithdrawFromSideChain)}
	mp := new(bloom.MerkleProof)

	datastore.AddMainChainTx(txHash1, genesisAddress, tx, mp)
	datastore.AddMainChainTx(txHash2, genesisAddress, tx, mp)
	datastore.AddMainChainTx(txHash3, genesisAddress, tx, mp)

	if ok, err := datastore.HasMainChainTx(txHash1, genesisAddress); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}
	if ok, err := datastore.HasMainChainTx(txHash2, genesisAddress); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}

	datastore.RemoveMainChainTxs([]string{txHash1, txHash2}, []string{genesisAddress, genesisAddress})

	ok, err := datastore.HasMainChainTx(txHash1, genesisAddress)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}
	ok, err = datastore.HasMainChainTx(txHash2, genesisAddress)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	if ok, err := datastore.HasMainChainTx(txHash3, genesisAddress); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}

	err = datastore.RemoveMainChainTx(txHash3, genesisAddress)
	if err != nil {
		t.Error("Remove main chain tx failed")
	}

	ok, err = datastore.HasMainChainTx(txHash3, genesisAddress)
	if err != nil {
		t.Error("Remove main chain tx error.")
	}
	if ok {
		t.Error("Remove main chain tx error.")
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_GetAllMainChainTxHashes(t *testing.T) {
	datastore, err := OpenMainChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash1 := "testHash1"
	txHash2 := "testHash2"
	txHash3 := "testHash3"
	genesisAddress := "genesis"

	tx := &Transaction{TxType: WithdrawFromSideChain, Payload: new(PayloadWithdrawFromSideChain)}

	mp := new(bloom.MerkleProof)

	datastore.AddMainChainTx(txHash1, genesisAddress, tx, mp)
	datastore.AddMainChainTx(txHash2, genesisAddress, tx, mp)
	datastore.AddMainChainTx(txHash3, genesisAddress, tx, mp)

	txHashes, genesisAddresses, err := datastore.GetAllMainChainTxHashes()
	if err != nil {
		t.Error("Get all main chain transactions error.")
	}
	if len(txHashes) != 3 || len(genesisAddresses) != 3 {
		t.Error("Get all main chain transactions error.")
	}
	for _, hash := range txHashes {
		if hash != txHash1 && hash != txHash2 && hash != txHash3 {
			t.Error("Get all main chain transactions error.")
		}
	}

	txHashes, genesisAddresses, txs, proofs, err := datastore.GetAllMainChainTxs()
	if err != nil {
		t.Error("Get all main chain transactions error.")
	}
	if len(txHashes) != 3 || len(genesisAddresses) != 3 || len(txs) != 3 || len(proofs) != 3 {
		t.Error("Get all main chain transactions error.")
	}
	for _, hash := range txHashes {
		if hash != txHash1 && hash != txHash2 && hash != txHash3 {
			t.Error("Get all main chain transactions error.")
		}
	}

	txs, proofs, err = datastore.GetMainChainTxsFromHashes([]string{txHash1, txHash2, txHash3}, genesisAddress)
	if err != nil {
		t.Error("Get main chain txs from hashes error.")
	}
	if len(txs) != 3 || len(proofs) != 3 {
		t.Error("Get main chain txs from hashes error.")
	}

	datastore.ResetDataStore()
}
