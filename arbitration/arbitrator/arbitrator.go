package arbitrator

import (
	"bytes"
	"encoding/binary"
	"errors"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/password"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
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

	InitAccount() error
	StartSpvModule() error

	//deposit
	ParseUserDepositTransactionInfo(txn *Transaction) ([]*DepositInfo, error)
	CreateDepositTransactions(proof bloom.MerkleProof, infoArray []*DepositInfo) map[*TransactionInfo]SideChain
	SendDepositTransactions(transactionInfoMap map[*TransactionInfo]SideChain)

	//withdraw
	CreateWithdrawTransaction(
		withdrawInfoMap []*WithdrawInfo, sideChain SideChain, sideTransactionHash string, mcFunc MainChainFunc) []*Transaction
	BroadcastWithdrawProposal(txns []*Transaction)
	SendWithdrawTransaction(txn *Transaction) (interface{}, error)
}

type ArbitratorImpl struct {
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

func (ar *ArbitratorImpl) GetComplainSolving() ComplainSolving {
	return nil
}

func (ar *ArbitratorImpl) Sign(content []byte) ([]byte, error) {
	mainAccount := ar.Keystore.MainAccount()

	return mainAccount.Sign(content)
}

func (ar *ArbitratorImpl) IsOnDutyOfMain() bool {
	pk, err := PublicKeyFromString(ArbitratorGroupSingleton.GetOnDutyArbitratorOfMain())
	if err != nil {
		return false
	}
	return crypto.Equal(pk, ar.GetPublicKey())
}

func (ar *ArbitratorImpl) IsOnDutyOfSide(sideChainKey string) bool {
	pk, err := PublicKeyFromString(ArbitratorGroupSingleton.GetOnDutyArbitratorOfSide(sideChainKey))
	if err != nil {
		return false
	}
	return crypto.Equal(pk, ar.GetPublicKey())
}

func (ar *ArbitratorImpl) GetArbitratorGroup() ArbitratorGroup {
	return ArbitratorGroupSingleton
}

func (ar *ArbitratorImpl) CreateWithdrawTransaction(
	withdrawInfoMap []*WithdrawInfo, sideChain SideChain, sideTransactionHash string, mcFunc MainChainFunc) []*Transaction {

	var result []*Transaction
	for _, info := range withdrawInfoMap {

		rateFloat := sideChain.GetRage()
		rate := common.Fixed64(rateFloat * 10000)
		amount := info.Amount * 10000 / rate
		withdrawTransaction, err := ar.mainChainImpl.CreateWithdrawTransaction(
			sideChain.GetKey(), info.TargetAddress, amount, sideTransactionHash, mcFunc)
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
	return ar.mainChainImpl.ParseUserDepositTransactionInfo(txn)
}

func (ar *ArbitratorImpl) CreateDepositTransactions(proof bloom.MerkleProof, infoArray []*DepositInfo) map[*TransactionInfo]SideChain {

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
		amount := info.Amount * rate / 10000
		txInfo, err := sideChain.CreateDepositTransaction(info.TargetAddress, proof, amount)
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
	if !ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfMain() {
		return
	}

	buf := new(bytes.Buffer)
	spvtxn.Serialize(buf)
	txBytes := buf.Bytes()

	r := bytes.NewReader(txBytes)
	txn := new(Transaction)
	txn.Deserialize(r)

	depositInfo, err := ar.ParseUserDepositTransactionInfo(txn)
	if err != nil {
		log.Error(err)
		return
	}

	transactionInfoMap := ar.CreateDepositTransactions(proof, depositInfo)
	ar.SendDepositTransactions(transactionInfoMap)
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

func (ar *ArbitratorImpl) InitAccount() error {
	passwd, err := password.GetAccountPassword()
	if err != nil {
		return errors.New("Get password error.")
	}

	ar.Keystore = NewKeystore()
	_, err = ar.Keystore.Open(string(passwd[:]))
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
	}

	account, err := sideauxpow.SelectAddress(sideauxpow.CurrentWallet)
	if err != nil {
		return err
	}
	if err = spvService.RegisterAccount(account); err != nil {
		return err
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
