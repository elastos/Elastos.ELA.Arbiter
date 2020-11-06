package mainchain

import (
	"testing"
	"time"

	abter "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
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

func (mcFunc *TestWithdrawFunc) GetWithdrawUTXOsByAmount(withdrawBank string) ([]*store.AddressUTXO, error) {
	var utxos []*store.AddressUTXO
	amount := common.Fixed64(10000000000)
	utxo := &store.AddressUTXO{
		Input: &types.Input{
			Previous: types.OutPoint{
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
	log.Init("./log", 1, 32, 64)
}

func TestCheckWithdrawTransaction(t *testing.T) {
	testLoopTimes := 10000

	//create data
	genesisAddress := "XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ"
	txHashStr := "dce8c840ce6e28516595737f5f81c10892c7a7ffbcae2289d57b52ecf4f529b2"

	var txInfos []*base.WithdrawTx
	var txHashes []string
	for i := 0; i < testLoopTimes; i++ {
		txBytes, _ := common.HexStringToBytes(txHashStr)
		txBytes[28] = byte(i)
		txBytes[29] = byte(i >> 8)
		txBytes[30] = byte(i >> 16)
		txBytes[31] = byte(i >> 24)
		txHash, _ := common.Uint256FromBytes(txBytes)

		//create withdraw transactionInfo
		txInfo := &base.WithdrawTx{
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

	store.DbCache.SideChainStore = &*scDataStore
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
	unsolvedTxs, _ := base.SubstractTransactionHashesAndBlockHeights(txHashes, blockHeights, []string{})
	if err != nil {
		t.Error("Get side chain txs from hashes failed")
	}
	endTime = time.Now()
	log.Info("GetSideChainTxsFromHashes Used time:", endTime.Sub(startTime).String())

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
