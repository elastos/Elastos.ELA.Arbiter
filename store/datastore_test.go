package store

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"

	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA.SPV/bloom"
	elacommon "github.com/elastos/Elastos.ELA/core/types/common"
	elatx "github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types/payload"
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

	txHash := "testHash"

	ok, err := datastore[0].HasSideChainTx(txHash)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}


	tx := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)

	buf := new(bytes.Buffer)
	tx.Serialize(buf)
	if err := datastore[0].AddSideChainTx(&base.SideChainTransaction{txHash, buf.Bytes(), 10}); err != nil {
		t.Error("Add side chain transaction error.")
	}

	ok, err = datastore[0].HasSideChainTx(txHash)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}
	DBNameSideChain := filepath.Join(DBDocumentNAME,
		config.Parameters.SideNodeList[0].Name+"_sideChainCache.db")
	datastore[0].ResetDataStore(DBNameSideChain)
}

func TestDataStoreImpl_AddSideChainTxs(t *testing.T) {
	datastore, err := OpenSideChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash1 := "testHash1"
	txHash2 := "testHash2"
	txHash3 := "testHash3"

	ok, err := datastore[0].HasSideChainTx(txHash1)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}
	ok, err = datastore[0].HasSideChainTx(txHash2)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}
	ok, err = datastore[0].HasSideChainTx(txHash3)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	tx := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)
	buf := new(bytes.Buffer)
	tx.Serialize(buf)
	err = datastore[0].AddSideChainTxs(
		[]*base.SideChainTransaction{
			&base.SideChainTransaction{txHash1, buf.Bytes(), 10},
			&base.SideChainTransaction{txHash2, buf.Bytes(), 10},
			&base.SideChainTransaction{txHash3, buf.Bytes(), 10},
		})
	if err != nil {
		t.Error("Add side chain transaction error.")
	}

	ok, err = datastore[0].HasSideChainTx(txHash1)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}
	ok, err = datastore[0].HasSideChainTx(txHash2)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}
	ok, err = datastore[0].HasSideChainTx(txHash3)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}

	DBNameSideChain := filepath.Join(DBDocumentNAME,
		config.Parameters.SideNodeList[0].Name+"_sideChainCache.db")
	datastore[0].ResetDataStore(DBNameSideChain)
}

func TestDataStoreImpl_RemoveSideChainTxs(t *testing.T) {
	datastore, err := OpenSideChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash := "testHash"
	tx := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)
	buf := new(bytes.Buffer)
	tx.Serialize(buf)

	txHash2 := "testHash2"
	tx2 := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)
	buf2 := new(bytes.Buffer)
	tx2.Serialize(buf2)

	datastore[0].AddSideChainTx(&base.SideChainTransaction{txHash, buf.Bytes(), 10})
	datastore[0].AddSideChainTx(&base.SideChainTransaction{txHash2, buf2.Bytes(), 10})

	if ok, err := datastore[0].HasSideChainTx(txHash); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}
	if ok, err := datastore[0].HasSideChainTx(txHash2); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}

	var removedHashes []string
	removedHashes = append(removedHashes, txHash)
	datastore[0].RemoveSideChainTxs(removedHashes)

	ok, err := datastore[0].HasSideChainTx(txHash)
	if err != nil {
		t.Error("Get side chain transaction error.")
	}
	if ok {
		t.Error("Should not have specified transaction.")
	}

	if ok, err := datastore[0].HasSideChainTx(txHash2); !ok || err != nil {
		t.Error("Should have specified transaction.")
	}
	DBNameSideChain := filepath.Join(DBDocumentNAME,
		config.Parameters.SideNodeList[0].Name+"_sideChainCache.db")
	datastore[0].ResetDataStore(DBNameSideChain)
}

func TestDataStoreImpl_GetAllSideChainTxHashes(t *testing.T) {
	datastore, err := OpenSideChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash := "testHash"
	txHash2 := "testHash2"

	txHash3 := "testHash3"

	tx := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)
	buf := new(bytes.Buffer)
	tx.Serialize(buf)
	datastore[0].AddSideChainTx(&base.SideChainTransaction{txHash, buf.Bytes(), 10})
	datastore[0].AddSideChainTx(&base.SideChainTransaction{txHash2, buf.Bytes(), 10})
	datastore[0].AddSideChainTx(&base.SideChainTransaction{txHash3, buf.Bytes(), 11})
	datastore[0].AddSideChainTx(&base.SideChainTransaction{txHash3, buf.Bytes(), 11})

	txHashes, err := datastore[0].GetAllSideChainTxHashes()
	if err != nil {
		t.Error("Get all side chain transactions error.")
	}
	if len(txHashes) != 3 {
		t.Error("Get all side chain transactions error.")
	}

	DBNameSideChain := filepath.Join(DBDocumentNAME,
		config.Parameters.SideNodeList[0].Name+"_sideChainCache.db")
	datastore[0].ResetDataStore(DBNameSideChain)
}

func TestDataStoreImpl_GetSideChainTxsFromHashes(t *testing.T) {
	datastore, err := OpenSideChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	txHash := "testHash"
	txHash2 := "testHash2"

	txHash3 := "testHash3"

	tx1 := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)
	buf1 := new(bytes.Buffer)
	tx1.Serialize(buf1)
	tx2  := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)
	buf2 := new(bytes.Buffer)
	tx2.Serialize(buf2)
	tx3 := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)
	buf3 := new(bytes.Buffer)
	tx3.Serialize(buf3)

	tx1.SetLockTime( 1)
	tx2.SetLockTime( 2)
	tx3.SetLockTime( 3)

	datastore[0].AddSideChainTx(&base.SideChainTransaction{txHash, buf1.Bytes(), 10})
	datastore[0].AddSideChainTx(&base.SideChainTransaction{txHash2, buf2.Bytes(), 10})
	datastore[0].AddSideChainTx(&base.SideChainTransaction{txHash3, buf3.Bytes(), 10})

	var txHashes []string
	txHashes = append(txHashes, txHash)
	txHashes = append(txHashes, txHash2)
	txHashes = append(txHashes, txHash3)

	txs, err := datastore[0].GetSideChainTxsFromHashes(txHashes)
	if err != nil {
		t.Error("Get all side chain transactions error.")
	}
	if len(txs) != 3 {
		t.Error("Get all side chain transactions error.")
	}

	DBNameSideChain := filepath.Join(DBDocumentNAME,
		config.Parameters.SideNodeList[0].Name+"_sideChainCache.db")
	datastore[0].ResetDataStore(DBNameSideChain)
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

	tx := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)
	mp := new(bloom.MerkleProof)
	if err := datastore.AddMainChainTx(&base.MainChainTransaction{txHash, genesisAddress, tx, mp}); err != nil {
		t.Error("Add main chain transaction error.")
	}

	ok, err = datastore.HasMainChainTx(txHash, genesisAddress)
	if err != nil {
		t.Error("Get main chain transaction error.")
	}
	if !ok {
		t.Error("Should have specified transaction.")
	}

	datastore.ResetDataStore(DBNameMainChain)
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

	tx := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)
	mp := new(bloom.MerkleProof)
	results, err := datastore.AddMainChainTxs(
		[]*base.MainChainTransaction{
			&base.MainChainTransaction{txHash1, genesisAddress1, tx, mp},
			&base.MainChainTransaction{txHash2, genesisAddress2, tx, mp},
			&base.MainChainTransaction{txHash3, genesisAddress3, tx, mp},
		})
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

	datastore.ResetDataStore(DBNameMainChain)
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

	tx := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)
	mp := new(bloom.MerkleProof)

	datastore.AddMainChainTx(&base.MainChainTransaction{txHash1, genesisAddress, tx, mp})
	datastore.AddMainChainTx(&base.MainChainTransaction{txHash2, genesisAddress, tx, mp})
	datastore.AddMainChainTx(&base.MainChainTransaction{txHash3, genesisAddress, tx, mp})

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

	datastore.ResetDataStore(DBNameMainChain)
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

	tx := elatx.CreateTransaction(
		elacommon.TxVersion09,
		elacommon.WithdrawFromSideChain,
		0,
		new(payload.WithdrawFromSideChain),
		[]*elacommon.Attribute{},
		[]*elacommon.Input{},
		[]*elacommon.Output{},
		0,
		[]*program.Program{},
	)

	mp := new(bloom.MerkleProof)
	datastore.AddMainChainTx(&base.MainChainTransaction{txHash1, genesisAddress, tx, mp})
	datastore.AddMainChainTx(&base.MainChainTransaction{txHash2, genesisAddress, tx, mp})
	datastore.AddMainChainTx(&base.MainChainTransaction{txHash3, genesisAddress, tx, mp})

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

	txs, err := datastore.GetAllMainChainTxs()
	if err != nil {
		t.Error("Get all main chain transactions error.")
	}
	if len(txs) != 3 {
		t.Error("Get all main chain transactions error.")
	}
	if txs[0].TransactionHash != txHash1 && txs[1].TransactionHash != txHash2 && txs[2].TransactionHash != txHash3 {
		t.Error("Get all main chain transactions error.")
	}

	spvTxs, err := datastore.GetMainChainTxsFromHashes([]string{txHash1, txHash2, txHash3}, genesisAddress)
	if err != nil {
		t.Error("Get main chain txs from hashes error.")
	}
	if len(spvTxs) != 3 {
		t.Error("Get main chain txs from hashes error.")
	}

	datastore.ResetDataStore(DBNameMainChain)
}
