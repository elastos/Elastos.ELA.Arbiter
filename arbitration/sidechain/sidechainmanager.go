package sidechain

import (
	"bytes"
	"errors"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
)

type SideChainManagerImpl struct {
	SideChains map[string]arbitrator.SideChain
}

func (sideManager *SideChainManagerImpl) OnReceivedRegisteredSideChain() error {
	txs, err := store.DbCache.RegisteredSideChainStore.GetAllRegisteredSideChainTxs()
	if err != nil {
		return errors.New("[OnReceivedRegisteredSideChain] %s" + err.Error())
	}

	if len(txs) == 0 {
		log.Info("No cached register sidechain transaction need to send")
		return nil
	}

	for _, transaction := range txs {
		if err != nil {
			return errors.New("[OnReceivedRegisteredSideChain] %s" + err.Error())
		}
		side := &SideChainImpl{
			Key: transaction.GenesisBlockAddress,
			CurrentConfig: &config.SideNodeConfig{
				Rpc: &config.RpcConfig{
					IpAddress:    transaction.RegisteredSideChain.IpAddr,
					HttpJsonPort: int(transaction.RegisteredSideChain.HttpJsonPort),
					User:         transaction.RegisteredSideChain.User,
					Pass:         transaction.RegisteredSideChain.Pass,
				},
				ExchangeRate:        1.0,
				GenesisBlockAddress: transaction.GenesisBlockAddress,
				GenesisBlock:        transaction.RegisteredSideChain.GenesisHash.String(),
				PowChain:            false,
			},
		}

		sideManager.AddChain(transaction.GenesisBlockAddress, side)
		SideChainAccountMonitor.AddListener(side)
		go SideChainAccountMonitor.SyncChainData(side.CurrentConfig)

		err = store.DbCache.RegisteredSideChainStore.RemoveRegisteredSideChainTx(transaction.TransactionHash, transaction.GenesisBlockAddress)
		if err != nil {
			return errors.New("[OnReceivedRegisteredSideChain] RemoveRegisteredSideChainTx %s" + err.Error())
		}
		writer := new(bytes.Buffer)
		transaction.RegisteredSideChain.Serialize(writer)
		err = store.FinishedTxsDbCache.AddSucceedRegisterTx(transaction.TransactionHash, transaction.GenesisBlockAddress, writer.Bytes())
		if err != nil {
			return errors.New("[OnReceivedRegisteredSideChain] AddSucceedRegisterTxs %s" + err.Error())
		}
	}

	return nil
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

func (sideManager *SideChainManagerImpl) CheckAndRemoveReturnDepositTransactionsFromDB() error {
	txHashes, err := store.FinishedTxsDbCache.GetAllReturnDepositTxs()
	if err != nil {
		return err
	}
	if len(txHashes) == 0 {
		return nil
	}
	receivedTxs, err := rpc.GetExistReturnDepositTransactions(txHashes)
	if err != nil {
		return err
	}

	if len(receivedTxs) != 0 {
		err = store.FinishedTxsDbCache.RemoveReturnDepositTxs(receivedTxs)
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

	_, ges, txData, err := store.FinishedTxsDbCache.GetRegisterTxs(true)
	if err != nil {
		log.Error("Error fetching data GetRegisterTxs ", err.Error())
		return
	}
	for i, transaction := range txData {
		side := &SideChainImpl{
			Key: ges[i],
			CurrentConfig: &config.SideNodeConfig{
				Rpc: &config.RpcConfig{
					IpAddress:    transaction.IpAddr,
					HttpJsonPort: int(transaction.HttpJsonPort),
					User:         transaction.User,
					Pass:         transaction.Pass,
				},
				ExchangeRate:        1.0,
				GenesisBlockAddress: ges[i],
				GenesisBlock:        transaction.GenesisHash.String(),
				PowChain:            false,
			},
		}
		sideChainManager.AddChain(ges[i], side)
		config.Parameters.SideNodeList = append(config.Parameters.SideNodeList, side.CurrentConfig)
	}

	currentArbitrator.SetSideChainManager(sideChainManager)
}
