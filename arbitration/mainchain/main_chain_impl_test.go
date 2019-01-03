package mainchain

import (
	"testing"
	"time"

	abter "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/sidechain"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
)

type TestWithdrawFunc struct {
}

func (mcFunc *TestWithdrawFunc) GetAvailableUtxos(withdrawBank string) ([]*store.AddressUTXO, error) {
	var utxos []*store.AddressUTXO
	amount := common.Fixed64(10000000000)
	utxo := &store.AddressUTXO{
		Input: &core.Input{
			Previous: core.OutPoint{
				TxID:  common.Uint256{},
				Index: 0,
			},
			Sequence: 0,
		},
		Amount:              &amount,
		GenesisBlockAddress: "XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ",
	}
	utxos = append(utxos, utxo)
	return utxos, nil
}

func (mcFunc *TestWithdrawFunc) GetMainNodeCurrentHeight() (uint32, error) {
	return 200, nil
}

func TestClientInit(t *testing.T) {
	config.InitMockConfig()
	log.Init(log.Path, log.Stdout)
}

func TestCheckWithdrawTransaction(t *testing.T) {
	testLoopTimes := 10000

	//create data
	genesisAddress := "XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ"
	address1 := "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"
	address2 := "ETcwuryQ3MfGWW1UyPrXx3UfEfAygBoM7J"
	address3 := "EbgLkYci91V9VMzyBnCs2kLYVuXHfCTkd6"
	destroyAddress := "0000000000000000000000000000000000"
	txHashStr := "dce8c840ce6e28516595737f5f81c10892c7a7ffbcae2289d57b52ecf4f529b2"
	blockHash := "964379552cbe499e01b6fb2a93776eacff0cb720e3d5ea73f3868d3398e4cd00"
	txId1 := common.Uint256{11}
	assetId := common.Uint256{12}
	amount1 := common.Fixed64(10000)
	amount2 := common.Fixed64(9000)
	amount3 := common.Fixed64(8000)
	amount4 := common.Fixed64(1000)

	inputInfo1 := InputInfo{TxID: txId1.String(), VOut: 0, Sequence: 0}
	outputInfo1 := OutputInfo{Value: amount1.String(), Index: 0, Address: destroyAddress, AssetID: assetId.String(), OutputLock: 0}
	outputInfo2 := OutputInfo{Value: amount2.String(), Index: 0, Address: destroyAddress, AssetID: assetId.String(), OutputLock: 0}
	outputInfo3 := OutputInfo{Value: amount4.String(), Index: 0, Address: address1, AssetID: assetId.String(), OutputLock: 0}

	var txInfos []*TransactionInfo
	var txHashes []string
	for i := 0; i < testLoopTimes; i++ {
		txBytes, _ := common.HexStringToBytes(txHashStr)
		txBytes[28] = byte(i)
		txBytes[29] = byte(i >> 8)
		txBytes[30] = byte(i >> 16)
		txBytes[31] = byte(i >> 24)
		txHash, _ := common.Uint256FromBytes(txBytes)

		//create withdraw transactionInfo
		txInfo := &TransactionInfo{
			TxId:           txHash.String(),
			Hash:           txHash.String(),
			Size:           0,
			VSize:          0,
			Version:        0,
			LockTime:       0,
			Inputs:         []InputInfo{inputInfo1},
			Outputs:        []OutputInfo{outputInfo1, outputInfo2, outputInfo3},
			BlockHash:      blockHash,
			Confirmations:  0,
			Time:           0,
			BlockTime:      0,
			TxType:         8,
			PayloadVersion: 0,
			Payload: &TransferCrossChainAssetInfo{
				CrossChainAddresses: []string{address2, address3},
				OutputIndexes:       []uint64{0, 1},
				CrossChainAmounts:   []common.Fixed64{amount2, amount3},
			},
			Attributes: nil,
			Programs:   nil,
		}

		txHashes = append(txHashes, txHash.String())
		txInfos = append(txInfos, txInfo)
		//log.Info(i, ":", txHash.String())
	}

	scDataStore, err := store.OpenSideChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}
	fhDataStore, err := store.OpenFinishedTxsDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	store.DbCache.SideChainStore = *scDataStore
	side := &sidechain.SideChainImpl{
		Key:           genesisAddress,
		CurrentConfig: config.Parameters.SideNodeList[0],
	}
	arbiter := abter.ArbitratorImpl{}
	mc := &MainChainImpl{&cs.DistributedNodeServer{}}
	arbiter.SetMainChain(mc)
	abter.ArbitratorGroupSingleton = &abter.ArbitratorGroupImpl{}

	startTime := time.Now()
	log.Info("Start time:", startTime.String())
	//onutxochanged will add tx into side chain db
	err = side.OnUTXOChanged(txInfos, 100)
	if err != nil {
		t.Error("OnUTXOChanged err")
	}
	endTime := time.Now()
	log.Info("OnUtxoChanged Used time:", endTime.Sub(startTime).String())

	startTime = time.Now()
	txHashes, blockHeights, err := store.DbCache.SideChainStore.GetAllSideChainTxHashesAndHeights(side.GetKey())
	if err != nil {
		t.Error("Get all withdraw txs failed")
	}
	endTime = time.Now()
	log.Info("GetAllSideChainTxHashesAndHeights Used time:", endTime.Sub(startTime).String())

	startTime = time.Now()
	unsolvedTxs, _ := SubstractTransactionHashesAndBlockHeights(txHashes, blockHeights, []string{})
	unsolvedTransactions, err := store.DbCache.SideChainStore.GetSideChainTxsFromHashes(unsolvedTxs)
	if err != nil {
		t.Error("Get side chain txs from hashes failed")
	}
	endTime = time.Now()
	log.Info("GetSideChainTxsFromHashes Used time:", endTime.Sub(startTime).String())

	startTime = time.Now()
	withdrawInfo, err := side.ParseUserWithdrawTransactionInfo(unsolvedTransactions)
	if err != nil {
		t.Error("Parse user withdraw transaction info failed")
	}
	transactions := arbiter.CreateWithdrawTransactions(withdrawInfo, side, txHashes, &TestWithdrawFunc{})
	if len(transactions) != 1 {
		t.Error("Create withdraw transaction failed")
	}
	endTime = time.Now()
	log.Info("CreateWithdrawTransactions Used time:", endTime.Sub(startTime).String())

	//if all txHashes has found on main chain
	startTime = time.Now()
	receivedTxs := unsolvedTxs
	err = store.DbCache.SideChainStore.RemoveSideChainTxs(receivedTxs)
	if err != nil {
		t.Error("remove side chain")
	}
	endTime = time.Now()
	log.Info("RemoveSideChainTxs time:", endTime.Sub(startTime).String())

	startTime = time.Now()
	err = fhDataStore.AddSucceedWithdrawTxs(receivedTxs)
	if err != nil {
		t.Error("add succeed withdrawTx failed")
	}
	endTime = time.Now()
	log.Info("AddSucceedWithdrawTxs time:", endTime.Sub(startTime).String())
	log.Info("End time:", endTime.String())

	store.DbCache.SideChainStore.ResetDataStore()
	fhDataStore.ResetDataStore()
}
