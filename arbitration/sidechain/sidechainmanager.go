package sidechain

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"io/ioutil"
	"strconv"
)

type SideChainManagerImpl struct {
	SideChains map[string]arbitrator.SideChain
}

func (sideManager *SideChainManagerImpl) OnReceivedRegisteredSideChain(info base.RegisterSidechainRpcInfo, currentHeight uint32) error {
	log.Info("Receive register sidechain rpc ", info.IpAddr, info.User, info.Pass, info.GenesisBlockHash)
	txs, err := store.DbCache.RegisteredSideChainStore.GetAllRegisteredSideChainTxs()
	if err != nil {
		return errors.New("[OnReceivedRegisteredSideChain] %s" + err.Error())
	}

	log.Info("Persisted register sidechain count ", len(txs))
	if len(txs) == 0 {
		log.Info("No cached register sidechain transaction need to send")
		return nil
	}

	for _, transaction := range txs {
		if transaction.RegisteredSideChain.GenesisHash.String() == info.GenesisBlockHash {
			exchangeRate, err := strconv.ParseFloat(transaction.RegisteredSideChain.ExchangeRate.String(), 64)
			if err != nil {
				return errors.New("[OnReceivedRegisteredSideChain] exchangeRate convert error %s" + err.Error())
			}
			side := &SideChainImpl{
				Key: transaction.GenesisBlockAddress,
				CurrentConfig: &config.SideNodeConfig{
					Rpc: &config.RpcConfig{
						IpAddress:    info.IpAddr,
						HttpJsonPort: info.Httpjsonport,
						User:         info.User,
						Pass:         info.Pass,
					},
					ExchangeRate:           exchangeRate,
					EffectiveHeight:        transaction.RegisteredSideChain.EffectiveHeight,
					GenesisBlockAddress:    transaction.GenesisBlockAddress,
					GenesisBlock:           transaction.RegisteredSideChain.GenesisHash.String(),
					PowChain:               false,
					SupportQuickRecharge:   true,
					SupportInvalidDeposit:  true,
					SupportInvalidWithdraw: true,
				},
			}

			sideManager.AddChain(transaction.GenesisBlockAddress, side)
			SideChainAccountMonitor.AddListener(side)
			if currentHeight >= transaction.RegisteredSideChain.EffectiveHeight {
				go SideChainAccountMonitor.SyncChainData(side.CurrentConfig, side)
			}
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

			// add registered side chain config to config.json
			config.Parameters.SideNodeList = append(config.Parameters.SideNodeList, side.CurrentConfig)
			data, _ := json.MarshalIndent(config.Parameters, "", "")
			_ = ioutil.WriteFile(config.DefaultConfigFilename, data, 0644)
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
	txHashes, err := store.DbCache.SideChainStore.GetAllReturnDepositTxs()
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
		err = store.DbCache.SideChainStore.RemoveReturnDepositTxs(receivedTxs)
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

func LoadRegisterSideChain(current arbitrator.Arbitrator) {
	_, ges, txData, err := store.FinishedTxsDbCache.GetRegisterTxs(true)
	if err != nil {
		log.Error("Error fetching data GetRegisterTxs ", err.Error())
		return
	}
	log.Info("Loading Register SideChain tx ", len(txData))
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
		current.GetSideChainManager().AddChain(ges[i], side)
		config.Parameters.SideNodeList = append(config.Parameters.SideNodeList, side.CurrentConfig)
	}
}
