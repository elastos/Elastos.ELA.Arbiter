package arbitrator

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA.Arbiter/wallet"

	. "github.com/elastos/Elastos.ELA.SPV/interface"
	scError "github.com/elastos/Elastos.ELA.SideChain/service"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
	. "github.com/elastos/Elastos.ELA/core"
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
	ParseUserDepositTransactionInfo(txn *Transaction, genesisAddress string) (*DepositInfo, error)
	CreateDepositTransactions(spvTxs []*SpvTransaction) map[*TransactionInfo]*DepositTxInfo
	SendDepositTransactions(transactionInfoMap map[*TransactionInfo]*DepositTxInfo)
	CreateAndSendDepositTransactions(spvTxs []*SpvTransaction, genesisAddresses string)

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
		//send deposit transaction
		depositTxs, err := ar.mainChainImpl.SyncMainChainCachedTxs()
		if err != nil {
			log.Warn(err)
		}
		for sideChain, txHashes := range depositTxs {
			ar.CreateAndSendDepositTransactionsInDB(sideChain, txHashes)
		}
		//send withdraw transaction
		for _, sc := range ar.sideChainManagerImpl.GetAllChains() {
			sc.SendCachedWithdrawTxs()
		}
		//send side chain pow transaction
		ar.sideChainManagerImpl.StartSideChainMining()
	} else {
		log.Info("[OnDutyArbitratorChanged] I became not on duty of main")
	}
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

func (ar *ArbitratorImpl) ParseUserDepositTransactionInfo(txn *Transaction, genesisAddress string) (*DepositInfo, error) {
	depositInfo, err := ar.mainChainImpl.ParseUserDepositTransactionInfo(txn, genesisAddress)
	if err != nil {
		return nil, err
	}

	return depositInfo, nil
}

type DepositTxInfo struct {
	mainChainTxHash string
	sideChain       SideChain
}

func (ar *ArbitratorImpl) CreateDepositTransactions(spvTxs []*SpvTransaction) map[*TransactionInfo]*DepositTxInfo {
	result := make(map[*TransactionInfo]*DepositTxInfo, 0)

	for i := 0; i < len(spvTxs); i++ {
		addr, err := spvTxs[i].DepositInfo.MainChainProgramHash.ToAddress()
		if err != nil {
			log.Warn("Invalid deposit program hash")
			return nil
		}
		sideChain, ok := ar.GetChain(addr)
		if !ok {
			log.Warn("Invalid deposit address")
			return nil
		}

		txInfo, err := sideChain.CreateDepositTransaction(spvTxs[i])
		if err != nil {
			log.Warn("Create deposit transaction failed")
			return nil
		}

		result[txInfo] = &DepositTxInfo{
			mainChainTxHash: spvTxs[i].MainChainTransaction.Hash().String(),
			sideChain:       sideChain,
		}
	}

	return result
}

func (ar *ArbitratorImpl) SendDepositTransactions(transactionInfoMap map[*TransactionInfo]*DepositTxInfo) {
	var failedTxInfos []*TransactionInfo
	var failedDepositTxBytes [][]byte
	var failedMainChainTxHashes []string
	var failedGenesisAddresses []string
	var succeedMainChainTxHashes []string
	var succeedGenesisAddresses []string
	for txInfo, depositTxInfo := range transactionInfoMap {
		resp, err := depositTxInfo.sideChain.SendTransaction(txInfo)
		if err != nil || resp.Error != nil && scError.ErrorCode(resp.Code) != scError.ErrDoubleSpend {
			log.Warn("Send deposit transaction failed, move to finished db, main chain tx hash:", depositTxInfo.mainChainTxHash)
			depositTxBytes, err := json.Marshal(txInfo)
			if err != nil {
				log.Error("Deposit transactionInfo to bytes failed")
				continue
			}
			failedTxInfos = append(failedTxInfos, txInfo)
			failedDepositTxBytes = append(failedDepositTxBytes, depositTxBytes)
			failedMainChainTxHashes = append(failedMainChainTxHashes, depositTxInfo.mainChainTxHash)
			failedGenesisAddresses = append(failedGenesisAddresses, depositTxInfo.sideChain.GetKey())
		} else if resp.Error == nil && resp.Result != nil || resp.Error != nil && scError.ErrorCode(resp.Code) == scError.ErrMainchainTxDuplicate {
			if resp.Error != nil {
				log.Info("Send deposit found transaction has been processed, move to finished db, main chain tx hash:", depositTxInfo.mainChainTxHash)
			} else {
				log.Info("Send deposit transaction succeed, move to finished db, main chain tx hash:", depositTxInfo.mainChainTxHash)
			}
			succeedMainChainTxHashes = append(succeedMainChainTxHashes, depositTxInfo.mainChainTxHash)
			succeedGenesisAddresses = append(succeedGenesisAddresses, depositTxInfo.sideChain.GetKey())
		} else {
			log.Warn("Send deposit transaction failed, need to resend, main chain tx hash:", depositTxInfo.mainChainTxHash)
		}
	}

	for i := 0; i < len(failedTxInfos); i++ {
		err := store.DbCache.MainChainStore.RemoveMainChainTxs(failedMainChainTxHashes, failedGenesisAddresses)
		if err != nil {
			log.Warn("Remove faild transaction from db failed")
		}
		err = store.FinishedTxsDbCache.AddFailedDepositTxs(failedMainChainTxHashes, failedGenesisAddresses, failedDepositTxBytes)
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
		Magic:           config.Parameters.MainNode.Magic,
		Foundation:      config.Parameters.MainNode.FoundationAddress,
		SeedList:        config.Parameters.MainNode.SpvSeedList,
		DefaultPort:     config.Parameters.MainNode.DefaultPort,
		MinPeersForSync: config.Parameters.MainNode.MinPeersForSync,
		MinOutbound:     config.Parameters.MainNode.MinOutbound,
		MaxConnections:  config.Parameters.MainNode.MaxConnections,
		OnRollback:      nil, // Not implemented yet
	}

	SpvService, err := NewSPVService(spvCfg)
	if err != nil {
		return err
	}

	for _, sideNode := range config.Parameters.SideNodeList {
		keystore, err := wallet.OpenKeystore(sideNode.KeystoreFile, passwd)
		if err != nil {
			return err
		}
		err = SpvService.RegisterTransactionListener(&AuxpowListener{ListenAddress: keystore.Address()})
		if err != nil {
			return err
		}
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

func (ar *ArbitratorImpl) CreateAndSendDepositTransactions(spvTxs []*SpvTransaction, genesisAddress string) {
	var txs []*SpvTransaction
	var finalTxHashes []string
	for i := 0; i < len(spvTxs); i++ {
		depositInfo, err := ar.ParseUserDepositTransactionInfo(spvTxs[i].MainChainTransaction, genesisAddress)
		if err != nil {
			log.Error(err)
			continue
		}
		txs = append(txs, &SpvTransaction{spvTxs[i].MainChainTransaction, spvTxs[i].Proof, depositInfo})
		finalTxHashes = append(finalTxHashes, spvTxs[i].MainChainTransaction.Hash().String())
	}

	transactionInfoMap := ar.CreateDepositTransactions(txs)
	ar.SendDepositTransactions(transactionInfoMap)
}

func (ar *ArbitratorImpl) CreateAndSendDepositTransactionsInDB(sideChain SideChain, txHashes []string) {
	spvTxs, err := store.DbCache.MainChainStore.GetMainChainTxsFromHashes(txHashes, sideChain.GetKey())
	if err != nil {
		return
	}

	ar.CreateAndSendDepositTransactions(spvTxs, sideChain.GetKey())
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
