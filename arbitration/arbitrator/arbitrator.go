package arbitrator

import (
	"bytes"
	"path/filepath"
	"sync"
	"time"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA.Arbiter/wallet"

	. "github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA/common"
	. "github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/crypto"
)

const (
	SCErrMainchainTxDuplicate int64 = 45013
	ErrInvalidMainchainTx     int64 = 45022
)

var SpvService SPVService

type Arbitrator interface {
	GetPublicKey() *crypto.PublicKey

	GetComplainSolving() ComplainSolving

	Sign(content []byte) ([]byte, error)

	IsOnDutyOfMain() bool
	GetArbitratorGroup() ArbitratorGroup
	GetSideChainManager() SideChainManager
	GetMainChain() MainChain

	InitAccount(passwd []byte) error
	StartSpvModule(passwd []byte) error

	//deposit
	SendDepositTransactions(spvTxs []*SpvTransaction, genesisAddress string)

	//withdraw
	CreateWithdrawTransactions(
		withdrawInfo *WithdrawInfo, sideChain SideChain, sideTransactionHash []string, mcFunc MainChainFunc) []*Transaction
	BroadcastWithdrawProposal(txns []*Transaction)
	SendWithdrawTransaction(txn *Transaction) (rpc.Response, error)

	CheckAndRemoveCrossChainTransactionsFromDBLoop()
}

type ArbitratorImpl struct {
	mainOnDutyMux *sync.Mutex
	isOnDuty      bool

	mainChainImpl        MainChain
	mainChainClientImpl  MainChainClient
	sideChainManagerImpl SideChainManager
	Keystore             Keystore
}

func (ar *ArbitratorImpl) GetSideChainManager() SideChainManager {
	return ar.sideChainManagerImpl
}

func (ar *ArbitratorImpl) GetPublicKey() *crypto.PublicKey {
	mainAccount := ar.Keystore.MainAccount()

	buf := new(bytes.Buffer)

	spvPublicKey := mainAccount.PublicKey()
	spvPublicKey.Serialize(buf)

	publicKey := new(crypto.PublicKey)
	publicKey.Deserialize(buf)

	return publicKey
}

func (ar *ArbitratorImpl) OnDutyArbitratorChanged(onDuty bool) {
	ar.mainOnDutyMux.Lock()
	ar.isOnDuty = onDuty
	ar.mainOnDutyMux.Unlock()

	if onDuty {
		log.Info("[OnDutyArbitratorChanged] I am on duty of main")
		ar.ProcessDepositTransactions()
		ar.processWithdrawTransactions()
		ar.ProcessSideChainPowTransaction()
	} else {
		log.Info("[OnDutyArbitratorChanged] I became not on duty of main")
	}
}

func (ar *ArbitratorImpl) ProcessDepositTransactions() {
	if err := ar.mainChainImpl.SyncMainChainCachedTxs(); err != nil {
		log.Warn(err)
	}
}

func (ar *ArbitratorImpl) processWithdrawTransactions() {
	for _, sc := range ar.sideChainManagerImpl.GetAllChains() {
		go sc.SendCachedWithdrawTxs()
	}
}

func (ar *ArbitratorImpl) ProcessSideChainPowTransaction() {
	ar.sideChainManagerImpl.StartSideChainMining()
}

func (ar *ArbitratorImpl) GetComplainSolving() ComplainSolving {
	return nil
}

func (ar *ArbitratorImpl) Sign(content []byte) ([]byte, error) {
	mainAccount := ar.Keystore.MainAccount()

	return mainAccount.Sign(content)
}

func (ar *ArbitratorImpl) IsOnDutyOfMain() bool {
	ar.mainOnDutyMux.Lock()
	defer ar.mainOnDutyMux.Unlock()
	return ar.isOnDuty
}

func (ar *ArbitratorImpl) GetArbitratorGroup() ArbitratorGroup {
	return ArbitratorGroupSingleton
}

func (ar *ArbitratorImpl) CreateWithdrawTransactions(withdrawInfo *WithdrawInfo, sideChain SideChain,
	sideTransactionHash []string, mcFunc MainChainFunc) []*Transaction {
	//todo divide into different transactions by the number of side chain transactions
	var result []*Transaction

	withdrawTransaction, err := ar.mainChainImpl.CreateWithdrawTransaction(sideChain, withdrawInfo, sideTransactionHash, mcFunc)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}
	if withdrawTransaction == nil {
		log.Warn("Created an empty withdraw transaction.")
		return nil
	}
	result = append(result, withdrawTransaction)

	return result
}

type DepositTxInfo struct {
	mainChainTxHash string
	sideChain       SideChain
}

func (ar *ArbitratorImpl) SendDepositTransactions(spvTxs []*SpvTransaction, genesisAddress string) {
	var failedMainChainTxHashes []string
	var failedGenesisAddresses []string
	var succeedMainChainTxHashes []string
	var succeedGenesisAddresses []string
	sideChain, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(genesisAddress)
	if !ok {
		log.Error("[SyncMainChainCachedTxs] Get side chain from genesis address failed, genesis address:", genesisAddress)
		return
	}
	for _, tx := range spvTxs {
		hash := tx.MainChainTransaction.Hash()
		resp, err := sideChain.SendTransaction(&hash)
		if err != nil || resp.Error != nil && resp.Code != ErrInvalidMainchainTx {
			log.Warn("Send deposit transaction failed, move to finished db, main chain tx hash:", hash.String())
			failedMainChainTxHashes = append(failedMainChainTxHashes, hash.String())
			failedGenesisAddresses = append(failedGenesisAddresses, genesisAddress)
		} else if resp.Error == nil && resp.Result != nil || resp.Error != nil && resp.Code == SCErrMainchainTxDuplicate {
			if resp.Error != nil {
				log.Info("Send deposit found transaction has been processed, move to finished db, main chain tx hash:", hash.String())
			} else {
				log.Info("Send deposit transaction succeed, move to finished db, main chain tx hash:", hash.String())
				if txHash, ok := resp.Result.(string); ok {
					log.Info("Send deposit transaction succeed, move to finished db, side chain tx hash:", txHash)
				} else {
					log.Info("Send deposit transaction, received invalid response")
				}
			}
			succeedMainChainTxHashes = append(succeedMainChainTxHashes, hash.String())
			succeedGenesisAddresses = append(succeedGenesisAddresses, genesisAddress)
		} else {
			log.Warn("Send deposit transaction failed, need to resend, main chain tx hash:", hash.String())
		}
	}

	for i := 0; i < len(failedMainChainTxHashes); i++ {
		err := store.DbCache.MainChainStore.RemoveMainChainTxs(failedMainChainTxHashes, failedGenesisAddresses)
		if err != nil {
			log.Warn("Remove faild transaction from db failed")
		}
		err = store.FinishedTxsDbCache.AddFailedDepositTxs(failedMainChainTxHashes, failedGenesisAddresses)
		if err != nil {
			log.Warn("Add faild transaction to finished db failed")
		}
	}
	for i := 0; i < len(succeedMainChainTxHashes); i++ {
		err := store.DbCache.MainChainStore.RemoveMainChainTxs(succeedMainChainTxHashes, succeedGenesisAddresses)
		if err != nil {
			log.Warn("Remove succeed deposit transaction from db failed")
		}
		err = store.FinishedTxsDbCache.AddSucceedDepositTxs(succeedMainChainTxHashes, succeedGenesisAddresses)
		if err != nil {
			log.Warn("Add succeed deposit transaction to finished db failed")
		}
	}
}

func (ar *ArbitratorImpl) BroadcastWithdrawProposal(txns []*Transaction) {
	for _, txn := range txns {
		err := ar.mainChainImpl.BroadcastWithdrawProposal(txn)
		if err != nil {
			log.Warn(err.Error())
		}
	}
}

func (ar *ArbitratorImpl) SendWithdrawTransaction(txn *Transaction) (rpc.Response, error) {
	content, err := ar.convertToTransactionContent(txn)
	if err != nil {
		return rpc.Response{}, err
	}

	log.Info("[Rpc-sendrawtransaction] Withdraw transaction to main chain：", config.Parameters.MainNode.Rpc.IpAddress, ":", config.Parameters.MainNode.Rpc.HttpJsonPort)
	resp, err := rpc.CallAndUnmarshalResponse("sendrawtransaction",
		rpc.Param("data", content), config.Parameters.MainNode.Rpc)
	if err != nil {
		return rpc.Response{}, err
	}

	return resp, nil
}

func (ar *ArbitratorImpl) ReceiveProposalFeedback(content []byte) error {
	return ar.mainChainImpl.ReceiveProposalFeedback(content)
}

func (ar *ArbitratorImpl) OnReceivedProposal(content []byte) error {
	return ar.mainChainClientImpl.OnReceivedProposal(content)
}

func (ar *ArbitratorImpl) GetChain(key string) (SideChain, bool) {
	return ar.sideChainManagerImpl.GetChain(key)
}

func (ar *ArbitratorImpl) GetAllChains() []SideChain {
	return ar.sideChainManagerImpl.GetAllChains()
}

func (ar *ArbitratorImpl) SetMainChain(chain MainChain) {
	ar.mainChainImpl = chain
}

func (ar *ArbitratorImpl) GetMainChain() MainChain {
	return ar.mainChainImpl
}

func (ar *ArbitratorImpl) SetMainChainClient(client MainChainClient) {
	ar.mainChainClientImpl = client
}

func (ar *ArbitratorImpl) SetSideChainManager(manager SideChainManager) {
	ar.sideChainManagerImpl = manager
}

func (ar *ArbitratorImpl) InitAccount(passwd []byte) error {
	ar.Keystore = NewKeystore()
	_, err := ar.Keystore.Open(string(passwd[:]))
	if err != nil {
		return err
	}
	accounts := ar.Keystore.GetAccounts()
	if len(accounts) <= 0 {
		ar.Keystore.NewAccount()
	}

	return nil
}

func (ar *ArbitratorImpl) StartSpvModule(passwd []byte) error {
	spvCfg := &Config{
		DataDir:        filepath.Join(config.DataPath, config.DataDir, config.SpvDir),
		Magic:          config.Parameters.MainNode.Magic,
		Foundation:     config.Parameters.MainNode.FoundationAddress,
		SeedList:       config.Parameters.MainNode.SpvSeedList,
		DefaultPort:    config.Parameters.MainNode.DefaultPort,
		MinOutbound:    config.Parameters.MainNode.MinOutbound,
		MaxConnections: config.Parameters.MainNode.MaxConnections,
		OnRollback:     nil, // Not implemented yet
	}

	log.Info("[StartSpvModule] new spv service:", spvCfg)

	var err error
	SpvService, err = NewSPVService(spvCfg)
	if err != nil {
		return err
	}

	for _, sideNode := range config.Parameters.SideNodeList {
		keystore, err := wallet.OpenKeystore(sideNode.KeystoreFile, passwd)
		if err != nil {
			return err
		}

		if sideNode.PowChain {
			log.Info("[StartSpvModule] register auxpow listener:", keystore.Address())
			auxpowListener := &AuxpowListener{ListenAddress: keystore.Address()}
			auxpowListener.start()
			err = SpvService.RegisterTransactionListener(auxpowListener)
			if err != nil {
				return err
			}
		}

		log.Info("[StartSpvModule] register dposit listener:", sideNode.GenesisBlockAddress)
		dpListener := &DepositListener{ListenAddress: sideNode.GenesisBlockAddress}
		dpListener.start()
		err = SpvService.RegisterTransactionListener(dpListener)
		if err != nil {
			return err
		}
	}

	go SpvService.Start()

	return nil
}

func (ar *ArbitratorImpl) convertToTransactionContent(txn *Transaction) (string, error) {
	buf := new(bytes.Buffer)
	err := txn.Serialize(buf)
	if err != nil {
		return "", err
	}
	content := common.BytesToHexString(buf.Bytes())
	return content, nil
}

func (ar *ArbitratorImpl) CheckAndRemoveCrossChainTransactionsFromDBLoop() {
	for {
		err := ar.mainChainImpl.CheckAndRemoveDepositTransactionsFromDB()
		if err != nil {
			log.Warn("Check and remove deposit transactions from db error:", err)
		}
		err = ar.GetSideChainManager().CheckAndRemoveWithdrawTransactionsFromDB()
		if err != nil {
			log.Warn("Check and remove withdraw transactions from db error:", err)
		}
		log.Info("Check and remove cross chain transactions from dbcache finished")
		time.Sleep(time.Millisecond * config.Parameters.ClearTransactionInterval)
	}
}
