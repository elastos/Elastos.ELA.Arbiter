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
	"github.com/elastos/Elastos.ELA/common"
	"io/ioutil"
	"strconv"
)

type SideChainManagerImpl struct {
	SideChains map[string]arbitrator.SideChain
}

func (sideManager *SideChainManagerImpl) OnReceivedRegisteredSideChain(info base.RegisterSidechainRpcInfo) error {
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
					Name:                   transaction.RegisteredSideChain.SideChainName,
					ExchangeRate:           exchangeRate,
					EffectiveHeight:        transaction.RegisteredSideChain.EffectiveHeight,
					GenesisBlockAddress:    transaction.GenesisBlockAddress,
					GenesisBlock:           transaction.RegisteredSideChain.GenesisHash.String(),
					PowChain:               false,
					SupportQuickRecharge:   true,
					SupportInvalidDeposit:  true,
					SupportInvalidWithdraw: true,
				},
				DoneSmallCrs: make(map[string]bool, 0),
			}

			// try create side chain db
			db, err := store.CreateSideChainDBByConfig(side.CurrentConfig)
			if err != nil {
				return errors.New("[OnReceivedRegisteredSideChain] CreateSideChainDBByConfig err:%s" + err.Error())
			}
			store.DbCache.SideChainStore = append(store.DbCache.SideChainStore, db)
			sideManager.AddChain(transaction.GenesisBlockAddress, side)
			SideChainAccountMonitor.AddListener(side)
			go SideChainAccountMonitor.SyncChainData(side.CurrentConfig, side, transaction.RegisteredSideChain.EffectiveHeight)
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
			var cf config.ConfigFile
			copyConfig(*config.Parameters.Configuration, &cf.ConfigFile)
			data, _ := json.MarshalIndent(cf, "", "")
			_ = ioutil.WriteFile(config.DefaultConfigFilename, data, 0644)
		}
	}

	return nil
}

func copyConfig(src config.Configuration, dest *config.Configuration) {
	dest.ActiveNet = src.ActiveNet
	dest.Magic = src.Magic
	dest.Version = src.Version
	dest.NodePort = src.NodePort
	dest.MainNode = src.MainNode
	for _, sn := range src.SideNodeList {
		genesisBytes, err := common.Uint256FromHexString(sn.GenesisBlock)
		if err != nil {
			log.Warn("Error parse genesisblock ", err.Error())
			continue
		}
		dest.SideNodeList = append(dest.SideNodeList, &config.SideNodeConfig{
			Rpc:                    sn.Rpc,
			ExchangeRate:           sn.ExchangeRate,
			EffectiveHeight:        sn.EffectiveHeight,
			GenesisBlockAddress:    sn.GenesisBlockAddress,
			GenesisBlock:           common.ToReversedString(*genesisBytes),
			KeystoreFile:           sn.KeystoreFile,
			MiningAddr:             sn.MiningAddr,
			PayToAddr:              sn.PayToAddr,
			PowChain:               sn.PowChain,
			SyncStartHeight:        sn.SyncStartHeight,
			SupportQuickRecharge:   sn.SupportQuickRecharge,
			SupportInvalidDeposit:  sn.SupportInvalidDeposit,
			SupportInvalidWithdraw: sn.SupportInvalidWithdraw,
		})
	}
	dest.SyncInterval = src.SyncInterval
	dest.HttpJsonPort = src.HttpJsonPort
	dest.HttpRestPort = src.HttpRestPort
	dest.PrintLevel = src.PrintLevel
	dest.SPVPrintLevel = src.SPVPrintLevel
	dest.MaxLogsSize = src.MaxLogsSize
	dest.MaxPerLogSize = src.MaxPerLogSize
	dest.SideChainMonitorScanInterval = src.SideChainMonitorScanInterval
	dest.ClearTransactionInterval = src.ClearTransactionInterval
	dest.MinOutbound = src.MinOutbound
	dest.MaxConnections = src.MaxConnections
	dest.MaxNodePerHost = src.MaxNodePerHost
	dest.SideAuxPowFee = src.SideAuxPowFee
	dest.MinThreshold = src.MinThreshold
	dest.SmallCrossTransferThreshold = src.SmallCrossTransferThreshold
	dest.DepositAmount = src.DepositAmount
	dest.CRCOnlyDPOSHeight = src.CRCOnlyDPOSHeight
	dest.CRClaimDPOSNodeStartHeight = src.CRClaimDPOSNodeStartHeight
	dest.NewP2PProtocolVersionHeight = src.NewP2PProtocolVersionHeight
	dest.DPOSNodeCrossChainHeight = src.DPOSNodeCrossChainHeight
	dest.MaxTxsPerWithdrawTx = src.MaxTxsPerWithdrawTx
	dest.OriginCrossChainArbiters = src.OriginCrossChainArbiters
	dest.CRCCrossChainArbiters = src.CRCCrossChainArbiters
	dest.RpcConfiguration = src.RpcConfiguration
	dest.DPoSNetAddress = src.DPoSNetAddress
	dest.ReturnDepositTransactionFee = src.ReturnDepositTransactionFee
	dest.NewCrossChainTransactionHeight = src.NewCrossChainTransactionHeight
	dest.ProcessInvalidWithdrawHeight = src.ProcessInvalidWithdrawHeight
	dest.WalletPath = src.WalletPath
	dest.ReturnCrossChainCoinStartHeight = src.ReturnCrossChainCoinStartHeight
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

	for _, s := range store.DbCache.SideChainStore {
		txHashes, err := s.GetAllSideChainTxHashes()
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
			err = s.RemoveSideChainTxs(receivedTxs)
			if err != nil {
				return err
			}

			err = store.FinishedTxsDbCache.AddSucceedWithdrawTxs(receivedTxs)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (sideManager *SideChainManagerImpl) CheckAndRemoveReturnDepositTransactionsFromDB() error {
	for _, s := range store.DbCache.SideChainStore {
		txHashes, err := s.GetAllReturnDepositTxs()
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
			err = s.RemoveReturnDepositTxs(receivedTxs)
			if err != nil {
				return err
			}
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
				Name:                transaction.SideChainName,
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
