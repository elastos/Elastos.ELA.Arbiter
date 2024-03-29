package arbitrator

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"path/filepath"
	"sync"
	"time"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	crypto2 "github.com/elastos/Elastos.ELA.Arbiter/arbitration/crypto"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	. "github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/common"
	it "github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
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

	InitAccount(client *account.Client)
	StartSpvModule() error

	//deposit
	SendDepositTransactions(spvTxs []*SpvTransaction, genesisAddress string)
	SendSmallCrossDepositTransactions(spvTxs []*SmallCrossTransaction, genesisAddress string)

	//withdraw
	CreateWithdrawTransactionV0(withdrawTxs []*WithdrawTx, sideChain SideChain,
		mcFunc MainChainFunc, mainChainHeight uint32) it.Transaction
	CreateWithdrawTransactionV1(withdrawTxs []*WithdrawTx, sideChain SideChain,
		mcFunc MainChainFunc, mainChainHeight uint32) it.Transaction
	CreateSchnorrWithdrawTransaction(withdrawTxs []*WithdrawTx, sideChain SideChain,
		mcFunc MainChainFunc, mainChainHeight uint32) it.Transaction

	CreateNFTDestroyTransaction(nftTxs []*NFTDestroyFromSideChainTx,
		sideChain SideChain, mcFunc MainChainFunc, mainChainHeight uint32) it.Transaction

	//failed deposit
	CreateFailedDepositTransaction(withdrawTxs []*FailedDepositTx,
		sideChain SideChain, mcFunc MainChainFunc) it.Transaction

	BroadcastWithdrawProposal(txn it.Transaction)

	SendWithdrawTransaction(txn it.Transaction) (rpc.Response, error)

	// schnorr withdraw
	BroadcastSchnorrWithdrawProposal2(txn it.Transaction)
	BroadcastSchnorrWithdrawProposal3(nonceHash common.Uint256, txn it.Transaction, pks [][]byte, e *big.Int)
	// schnorr crypto
	GetSchnorrR() (k0 *big.Int, rx *big.Int, ry *big.Int, px *big.Int, py *big.Int, err error)
	GetSchnorrS(e *big.Int) *big.Int

	BroadcastSidechainIllegalData(data *payload.SidechainIllegalData)

	CheckAndRemoveCrossChainTransactionsFromDBLoop()
}

type ArbitratorImpl struct {
	mainOnDutyMux *sync.Mutex
	isOnDuty      bool

	mainChainImpl        MainChain
	mainChainClientImpl  MainChainClient
	sideChainManagerImpl SideChainManager
	client               *account.Client
}

func (ar *ArbitratorImpl) GetSideChainManager() SideChainManager {
	return ar.sideChainManagerImpl
}

func (ar *ArbitratorImpl) GetPublicKey() *crypto.PublicKey {
	mainAccount := ar.client.GetMainAccount()

	return mainAccount.PubKey()
}

func (ar *ArbitratorImpl) OnDutyArbitratorChanged(onDuty bool) {
	ar.mainOnDutyMux.Lock()
	ar.isOnDuty = onDuty
	ar.mainOnDutyMux.Unlock()

	if onDuty {
		log.Info("[OnDutyArbitratorChanged] I am on duty of main")
		var currentHeight = store.DbCache.MainChainStore.CurrentHeight(
			store.QueryHeightCode)
		ar.mainChainImpl.Reset()
		ar.ProcessDepositTransactions()
		ar.processWithdrawTransactions(currentHeight)
		ar.processNFTDestroyTransactions(currentHeight)
		ar.processReturnDepositTransactions()
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

func (ar *ArbitratorImpl) processWithdrawTransactions(currentHeight uint32) {
	for _, sc := range ar.sideChainManagerImpl.GetAllChains() {
		go sc.SendCachedWithdrawTxs(currentHeight)
	}
}

func (ar *ArbitratorImpl) processNFTDestroyTransactions(currentHeight uint32) {
	for _, sc := range ar.sideChainManagerImpl.GetAllChains() {
		go sc.SendCachedNFTDestroyTxs(currentHeight)
	}
}

func (ar *ArbitratorImpl) processReturnDepositTransactions() {
	currentHeight := ArbitratorGroupSingleton.GetCurrentHeight()
	if currentHeight < config.Parameters.ReturnCrossChainCoinStartHeight {
		return
	}

	for _, sc := range ar.sideChainManagerImpl.GetAllChains() {
		go sc.SendCachedReturnDepositTxs()
	}
}

func (ar *ArbitratorImpl) ProcessSideChainPowTransaction() {
	ar.sideChainManagerImpl.StartSideChainMining()
}

func (ar *ArbitratorImpl) GetComplainSolving() ComplainSolving {
	return nil
}

func (ar *ArbitratorImpl) Sign(content []byte) ([]byte, error) {
	mainAccount := ar.client.GetMainAccount()

	return mainAccount.Sign(content)
}

func (ar *ArbitratorImpl) GetSchnorrR() (k0 *big.Int, rx *big.Int, ry *big.Int, px *big.Int, py *big.Int, err error) {
	mainAccount := ar.client.GetMainAccount()
	privKey := new(big.Int).SetBytes(mainAccount.PrivateKey)
	return crypto2.GetR(privKey)
}

func (ar *ArbitratorImpl) GetSchnorrS(e *big.Int) *big.Int {
	mainAccount := ar.client.GetMainAccount()
	privKey := new(big.Int).SetBytes(mainAccount.PrivateKey)
	return crypto2.GetEMulPrivateKey(privKey, e)
}

func (ar *ArbitratorImpl) IsOnDutyOfMain() bool {
	ar.mainOnDutyMux.Lock()
	defer ar.mainOnDutyMux.Unlock()
	return ar.isOnDuty
}

func (ar *ArbitratorImpl) GetArbitratorGroup() ArbitratorGroup {
	return ArbitratorGroupSingleton
}

func (ar *ArbitratorImpl) CreateFailedDepositTransaction(withdrawTxs []*FailedDepositTx,
	sideChain SideChain, mcFunc MainChainFunc) it.Transaction {
	ftx, err := ar.mainChainImpl.CreateFailedDepositTransaction(
		sideChain, withdrawTxs, mcFunc)
	if err != nil {
		log.Warn("[CreateFailedDepositTransaction]" + err.Error())
		return nil
	}
	if ftx == nil {
		log.Warn("[CreateFailedDepositTransaction] failed to create an failed deposit transaction.")
		return nil
	}
	log.Infof("failed deposit transaction %v", ftx)

	log.Info("[CreateFailedDepositTransaction] succeed")

	return ftx
}

func (ar *ArbitratorImpl) CreateWithdrawTransactionV0(withdrawTxs []*WithdrawTx,
	sideChain SideChain, mcFunc MainChainFunc, mainChainHeight uint32) it.Transaction {

	withdrawTransaction, err := ar.mainChainImpl.CreateWithdrawTransactionV0(
		sideChain, withdrawTxs, mcFunc)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}

	if withdrawTransaction == nil {
		log.Warn("Created an empty withdraw transaction v0.")
		return nil
	}

	return withdrawTransaction
}

func (ar *ArbitratorImpl) CreateWithdrawTransactionV1(withdrawTxs []*WithdrawTx,
	sideChain SideChain, mcFunc MainChainFunc, mainChainHeight uint32) it.Transaction {

	withdrawTransaction, err := ar.mainChainImpl.CreateWithdrawTransactionV1(
		sideChain, withdrawTxs, mcFunc)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}

	if withdrawTransaction == nil {
		log.Warn("Created an empty withdraw transaction v1.")
		return nil
	}

	return withdrawTransaction
}

func (ar *ArbitratorImpl) CreateSchnorrWithdrawTransaction(withdrawTxs []*WithdrawTx,
	sideChain SideChain, mcFunc MainChainFunc, mainChainHeight uint32) it.Transaction {

	withdrawTransaction, err := ar.mainChainImpl.CreateSchnorrWithdrawTransaction(
		sideChain, withdrawTxs, mcFunc)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}

	if withdrawTransaction == nil {
		log.Warn("Created an empty Schnorr withdraw transaction.")
		return nil
	}
	return withdrawTransaction
}

func (ar *ArbitratorImpl) CreateNFTDestroyTransaction(nftTxs []*NFTDestroyFromSideChainTx,
	sideChain SideChain, mcFunc MainChainFunc, mainChainHeight uint32) it.Transaction {

	nftDestroyTXs, err := ar.mainChainImpl.CreateNFTDestroyFromSideChainTx(
		sideChain, nftTxs, mcFunc, mainChainHeight)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}

	if nftDestroyTXs == nil {
		log.Warn("Created an empty NFT Destroy Transaction")
		return nil
	}

	return nftDestroyTXs
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
		log.Error("[SendDepositTransactions] Get side chain from genesis address failed, genesis address:", genesisAddress)
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

func (ar *ArbitratorImpl) SendSmallCrossDepositTransactions(knownTx []*SmallCrossTransaction, genesisAddress string) {
	sideChain, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(genesisAddress)
	if !ok {
		log.Error("[SendSmallCrossDepositTransactions] Get side chain from genesis address failed, genesis address:", genesisAddress)
		return
	}
	for _, tx := range knownTx {
		buf := new(bytes.Buffer)
		tx.MainTx.Serialize(buf)
		rawTx := hex.EncodeToString(buf.Bytes())
		signature := tx.Signature
		hash := tx.MainTx.Hash().String()
		if sideChain.IsSendSmallCrxTx(hash) {
			continue
		}
		_, err := sideChain.SendSmallCrossTransaction(rawTx, signature, hash)
		if err != nil {
			log.Info("Send deposit transaction Error", err.Error())
		}
	}
}

func (ar *ArbitratorImpl) BroadcastSchnorrWithdrawProposal2(txn it.Transaction) {
	err := ar.mainChainImpl.BroadcastSchnorrWithdrawProposal2(txn)
	if err != nil {
		log.Warn(err.Error())
	}
}

func (ar *ArbitratorImpl) BroadcastSchnorrWithdrawProposal3(
	nonceHash common.Uint256, txn it.Transaction, pks [][]byte, e *big.Int) {
	err := ar.mainChainImpl.BroadcastSchnorrWithdrawProposal3(nonceHash, txn, pks, e)
	if err != nil {
		log.Warn(err.Error())
	}
}

func (ar *ArbitratorImpl) BroadcastWithdrawProposal(txn it.Transaction) {
	err := ar.mainChainImpl.BroadcastWithdrawProposal(txn)
	if err != nil {
		log.Warn(err.Error())
	}
}

func (ar *ArbitratorImpl) BroadcastSidechainIllegalData(data *payload.SidechainIllegalData) {
	if err := ar.mainChainImpl.BroadcastSidechainIllegalData(data); err != nil {
		log.Warn(err.Error())
	}
}

func (ar *ArbitratorImpl) SendWithdrawTransaction(txn it.Transaction) (rpc.Response, error) {
	content, err := ar.convertToTransactionContent(txn)
	if err != nil {
		return rpc.Response{}, err
	}

	log.Info("[Rpc-sendrawtransaction] Withdraw transaction to main chain：",
		config.Parameters.MainNode.Rpc.IpAddress, ":", config.Parameters.MainNode.Rpc.HttpJsonPort)
	resp, err := rpc.CallAndUnmarshalResponse("sendrawtransaction",
		rpc.Param("data", content), config.Parameters.MainNode.Rpc)
	if err != nil {
		log.Error("[Rpc-sendrawtransaction] Withdraw transaction to main "+
			"chain error:", err)
		return rpc.Response{}, err
	}

	return resp, nil
}

func (ar *ArbitratorImpl) ReceiveProposalFeedback(content []byte) error {
	return ar.mainChainImpl.ReceiveProposalFeedback(content)
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

func (ar *ArbitratorImpl) InitAccount(client *account.Client) {
	ar.client = client
}

func (ar *ArbitratorImpl) StartSpvModule() error {
	params := config.GetSpvChainParams()
	spvCfg := &Config{
		DataDir:        filepath.Join(config.DataPath, config.DataDir, config.SpvDir),
		ChainParams:    params,
		PermanentPeers: config.Parameters.MainNode.SpvSeedList,
		NodeVersion:    config.NodePrefix + config.Version,
	}

	var err error
	SpvService, err = NewSPVService(spvCfg)
	if err != nil {
		return err
	}

	for _, sideNode := range config.Parameters.SideNodeList {
		if sideNode.PowChain {
			log.Info("[StartSpvModule] register auxpow listener:", sideNode.MiningAddr)
			auxpowListener := &AuxpowListener{ListenAddress: sideNode.MiningAddr}
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

func (ar *ArbitratorImpl) convertToTransactionContent(txn it.Transaction) (string, error) {
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
		err = ar.GetSideChainManager().CheckAndRemoveReturnDepositTransactionsFromDB()
		if err != nil {
			log.Warn("Check and remove return deposit transactions from db error:", err)
		}
		log.Info("Check and remove cross chain transactions from dbcache finished")
		time.Sleep(time.Millisecond * config.Parameters.ClearTransactionInterval)
	}
}
