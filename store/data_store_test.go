package store

import (
	"os"
	"testing"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
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

	ok, err := datastore.HashSideChainTx(txHash)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	if err := datastore.AddSideChainTx(txHash, genesisBlockAddress); err != nil {
		t.Error("Add side chain transaction error.")
	}

	ok, err = datastore.HashSideChainTx(txHash)
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

	genesisBlockAddress2 := "testAddress2"
	txHash2 := "testHash2"

	datastore.AddSideChainTx(txHash, genesisBlockAddress)
	datastore.AddSideChainTx(txHash2, genesisBlockAddress2)

	if ok, err := datastore.HashSideChainTx(txHash); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}
	if ok, err := datastore.HashSideChainTx(txHash2); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}

	var removedHashes []string
	removedHashes = append(removedHashes, txHash)
	datastore.RemoveSideChainTxs(removedHashes)

	ok, err := datastore.HashSideChainTx(txHash)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	if ok, err := datastore.HashSideChainTx(txHash2); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}

	datastore.ResetDataStore()
}

func TestDataStoreImpl_GetAllSideChainTxs(t *testing.T) {
	datastore, err := OpenDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress := "testAddress"
	txHash := "testHash"
	txHash2 := "testHash2"

	genesisBlockAddress2 := "testAddress2"
	txHash3 := "testHash3"

	datastore.AddSideChainTx(txHash, genesisBlockAddress)
	datastore.AddSideChainTx(txHash2, genesisBlockAddress)
	datastore.AddSideChainTx(txHash3, genesisBlockAddress2)

	txHashes, err := datastore.GetAllSideChainTxs(genesisBlockAddress)
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

	if err := datastore.AddMainChainTx(txHash); err != nil {
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

	datastore.AddMainChainTx(txHash)
	datastore.AddMainChainTx(txHash2)

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

func TestDataStoreImpl_GetAllMainChainTxs(t *testing.T) {
	datastore, err := OpenDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash := "testHash"
	txHash2 := "testHash2"
	txHash3 := "testHash3"

	datastore.AddMainChainTx(txHash)
	datastore.AddMainChainTx(txHash2)
	datastore.AddMainChainTx(txHash3)

	txHashes, err := datastore.GetAllMainChainTxs()
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

	datastore.ResetDataStore()
}
