package arbitrator

import (
	"bytes"
	"encoding/binary"
	"sync"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA.Arbiter/wallet"
	. "github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
	"github.com/elastos/Elastos.ELA/bloom"
	. "github.com/elastos/Elastos.ELA/core"
)

var spvService SPVService

type Arbitrator interface {
	GetPublicKey() *crypto.PublicKey

	GetComplainSolving() ComplainSolving

	Sign(content []byte) ([]byte, error)

	IsOnDutyOfMain() bool
	IsOnDutyOfSide(sideChainKey string) bool
	GetArbitratorGroup() ArbitratorGroup
	GetSideChainManager() SideChainManager

	InitAccount(passwd []byte) error
	StartSpvModule() error

	//deposit
	ParseUserDepositTransactionInfo(txn *Transaction) ([]*DepositInfo, error)
	CreateDepositTransactions(proof bloom.MerkleProof, mainChainTransaction *Transaction, infoArray []*DepositInfo) map[*TransactionInfo]SideChain
	SendDepositTransactions(transactionInfoMap map[*TransactionInfo]SideChain)

	//withdraw
	CreateWithdrawTransactions(
		withdrawInfoMap []*WithdrawInfo, sideChain SideChain, sideTransactionHash string, mcFunc MainChainFunc) []*Transaction
	BroadcastWithdrawProposal(txns []*Transaction)
	SendWithdrawTransaction(txn *Transaction) (interface{}, error)
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
		txs, proofs, err := ar.mainChainImpl.SyncMainChainCachedTxs()
		if err != nil {
			log.Warn(err)
		}

		for i := range txs {
			ar.createAndSendDepositTransaction(proofs[i], txs[i])
		}
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

func (ar *ArbitratorImpl) IsOnDutyOfSide(sideChainKey string) bool {
	chain, ok := ar.sideChainManagerImpl.GetChain(sideChainKey)
	if !ok {
		return false
	}

	return chain.IsOnDuty()
}

func (ar *ArbitratorImpl) GetArbitratorGroup() ArbitratorGroup {
	return ArbitratorGroupSingleton
}

func (ar *ArbitratorImpl) CreateWithdrawTransactions(withdrawInfoMap []*WithdrawInfo, sideChain SideChain,
	sideTransactionHash string, mcFunc MainChainFunc) []*Transaction {

	var result []*Transaction
	for _, info := range withdrawInfoMap {

		rateFloat := sideChain.GetRage()
		rate := common.Fixed64(rateFloat * 10000)
		amount := info.Amount * 10000 / rate
		crossChainAmount := info.CrossChainAmount * 10000 / rate
		withdrawTransaction, err := ar.mainChainImpl.CreateWithdrawTransaction(
			sideChain.GetKey(), info.TargetAddress, amount, crossChainAmount, sideTransactionHash, mcFunc)
		if err != nil {
			log.Warn(err.Error())
			continue
		}
		if withdrawTransaction == nil {
			log.Warn("Created an empty withdraw transaction.")
			continue
		}

		result = append(result, withdrawTransaction)
	}

	return result
}

func (ar *ArbitratorImpl) ParseUserDepositTransactionInfo(txn *Transaction) ([]*DepositInfo, error) {
	depositInfo, err := ar.mainChainImpl.ParseUserDepositTransactionInfo(txn)
	if err != nil {
		return nil, err
	}

	return depositInfo, nil
}

func (ar *ArbitratorImpl) CreateDepositTransactions(proof bloom.MerkleProof, mainChainTransaction *Transaction,
	infoArray []*DepositInfo) map[*TransactionInfo]SideChain {

	result := make(map[*TransactionInfo]SideChain, len(infoArray))
	for _, info := range infoArray {
		addr, err := info.MainChainProgramHash.ToAddress()
		if err != nil {
			log.Warn("Invalid deposit address.")
			continue
		}
		sideChain, ok := ar.GetChain(addr)
		if !ok {
			log.Warn("Invalid deposit address.")
			continue
		}

		rateFloat := sideChain.GetRage()
		rate := common.Fixed64(rateFloat * 10000)
		amount := info.CrossChainAmount * rate / 10000
		txInfo, err := sideChain.CreateDepositTransaction(info.TargetAddress, proof, mainChainTransaction, amount)
		if err != nil {
			log.Error(err)
			continue
		}

		result[txInfo] = sideChain
	}
	return result
}

func (ar *ArbitratorImpl) SendDepositTransactions(transactionInfoMap map[*TransactionInfo]SideChain) {
	for info, sideChain := range transactionInfoMap {
		err := sideChain.SendTransaction(info)
		if err != nil {
			log.Warn(err.Error())
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

func (ar *ArbitratorImpl) SendWithdrawTransaction(txn *Transaction) (interface{}, error) {
	content, err := ar.convertToTransactionContent(txn)
	if err != nil {
		return nil, err
	}

	result, err := rpc.CallAndUnmarshal("sendrawtransaction",
		rpc.Param("Data", content), config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (ar *ArbitratorImpl) ReceiveProposalFeedback(content []byte) error {
	return ar.mainChainImpl.ReceiveProposalFeedback(content)
}

func (ar *ArbitratorImpl) Type() TransactionType {
	return TransferCrossChainAsset
}

func (ar *ArbitratorImpl) Confirmed() bool {
	return true
}

func (ar *ArbitratorImpl) Notify(proof bloom.MerkleProof, spvtxn Transaction) {

	if ok, _ := store.DbCache.HashMainChainTx(spvtxn.Hash().String()); ok {
		return
	}

	if err := store.DbCache.AddMainChainTx(spvtxn.Hash().String(), &spvtxn, &proof); err != nil {
		log.Error("AddMainChainTx error, txHash:", spvtxn.Hash().String())
		return
	}

	if !ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfMain() {
		return
	}

	ar.createAndSendDepositTransaction(&proof, &spvtxn)
}

func (ar *ArbitratorImpl) Rollback(height uint32) {
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

func (ar *ArbitratorImpl) StartSpvModule() error {
	publicKey := ar.Keystore.MainAccount().PublicKey()
	publicKeyBytes, err := publicKey.EncodePoint(true)
	if err != nil {
		return err
	}

	spvService = NewSPVService(binary.LittleEndian.Uint64(publicKeyBytes), config.Parameters.MainNode.SpvSeedList)

	for _, sideNode := range config.Parameters.SideNodeList {
		if err = spvService.RegisterAccount(sideNode.GenesisBlockAddress); err != nil {
			return err
		}

		keystoreFile := sideauxpow.KeystoreDict[sideNode.GenesisBlock]
		keystore, err := wallet.OpenKeystore(keystoreFile, sideauxpow.Passwd)
		if err != nil {
			return err
		}

		account := keystore.Address()
		if err = spvService.RegisterAccount(account); err != nil {
			return err
		}
	}

	spvService.RegisterTransactionListener(ar)
	spvService.RegisterTransactionListener(&auxpowListener)

	go func() {
		if err = spvService.Start(); err != nil {
			log.Error("spvService start failed ：", err)
		}
	}()

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

func (ar *ArbitratorImpl) createAndSendDepositTransaction(proof *bloom.MerkleProof, spvtxn *Transaction) {
	depositInfo, err := ar.ParseUserDepositTransactionInfo(spvtxn)
	if err != nil {
		log.Error(err)
		return
	}

	transactionInfoMap := ar.CreateDepositTransactions(*proof, spvtxn, depositInfo)
	ar.SendDepositTransactions(transactionInfoMap)

	spvService.SubmitTransactionReceipt(spvtxn.Hash())
}
