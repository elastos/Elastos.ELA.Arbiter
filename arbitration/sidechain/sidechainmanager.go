package sidechain

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
)

type SideChainManagerImpl struct {
	SideChains map[string]arbitrator.SideChain
}

func (sideManager *SideChainManagerImpl) AddChain(key string, chain arbitrator.SideChain) {
	sideManager.SideChains[key] = chain
}

func (sideManager *SideChainManagerImpl) GetChain(key string) (arbitrator.SideChain, bool) {
	elem, ok := sideManager.SideChains[key]
	return elem, ok
}

func (sideManager *SideChainManagerImpl) GetAllChains() []arbitrator.SideChain {
	var chains []arbitrator.SideChain
	for _, v := range sideManager.SideChains {
		chains = append(chains, v)
	}
	return chains
}

func (sideManager *SideChainManagerImpl) StartSideChainMining() {
	for _, sc := range sideManager.SideChains {
		go sc.StartSideChainMining()
	}
}

func (sideManager *SideChainManagerImpl) CheckAndRemoveWithdrawTransactionsFromDB() error {
	txHashes, err := store.DbCache.SideChainStore.GetAllSideChainTxHashes()
	if err != nil {
		return err
	}
	if len(txHashes) == 0 {
		return nil
	}
	receivedTxs, err := rpc.GetExistWithdrawTransactions(txHashes)
	if err != nil {
		return err
	}

	if len(receivedTxs) != 0 {
		err = store.DbCache.SideChainStore.RemoveSideChainTxs(receivedTxs)
		if err != nil {
			return err
		}

		err = store.FinishedTxsDbCache.AddSucceedWithdrawTxs(receivedTxs)
		if err != nil {
			return err
		}
	}

	return nil
}

func Init() {
	currentArbitrator, ok := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().(*arbitrator.ArbitratorImpl)
	if !ok {
		return
	}

	sideChainManager := &SideChainManagerImpl{SideChains: make(map[string]arbitrator.SideChain)}
	for _, sideConfig := range config.Parameters.SideNodeList {
		side := &SideChainImpl{
			Key:           sideConfig.GenesisBlockAddress,
			CurrentConfig: sideConfig,
			DoneSmallCrs:  make(map[string]bool),
		}

		sideChainManager.AddChain(sideConfig.GenesisBlockAddress, side)
		log.Infof("Init Sidechain config ", side.Key, side.CurrentConfig.SupportQuickRecharge, side.CurrentConfig.GenesisBlock)
	}
	currentArbitrator.SetSideChainManager(sideChainManager)
}
