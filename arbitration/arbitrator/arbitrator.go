package arbitrator

import (
	"bytes"
	"encoding/binary"
	"errors"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/common"
	"github.com/elastos/Elastos.ELA.Arbiter/common/config"
	"github.com/elastos/Elastos.ELA.Arbiter/common/log"
	"github.com/elastos/Elastos.ELA.Arbiter/common/password"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
	"github.com/elastos/Elastos.ELA.Arbiter/crypto"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	. "github.com/elastos/Elastos.ELA.SPV/interface"
	spv "github.com/elastos/Elastos.ELA.SPV/interface"
	utcore "github.com/elastos/Elastos.ELA.Utility/core"
)

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
	ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error)
	CreateDepositTransactions(proof spv.Proof, infoArray []*DepositInfo) map[*TransactionInfo]SideChain
	SendDepositTransactions(transactionInfoMap map[*TransactionInfo]SideChain)

	//withdraw
	CreateWithdrawTransaction(
		withdrawInfoMap []*WithdrawInfo, sideChain SideChain, sideTransactionHash string, mcFunc MainChainFunc) []*tx.Transaction
	BroadcastWithdrawProposal(txns []*tx.Transaction)
	SendWithdrawTransaction(txn *tx.Transaction) (interface{}, error)
}

type ArbitratorImpl struct {
	mainChainImpl        MainChain
	mainChainClientImpl  MainChainClient
	sideChainManagerImpl SideChainManager
	spvService           SPVService
	Keystore             Keystore
}

func (ar *ArbitratorImpl) GetSideChainManager() SideChainManager {
	return ar.GetSideChainManager()
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
	pk := crypto.PublicKey{}
	pk.FromString(ArbitratorGroupSingleton.GetOnDutyArbitratorOfMain())
	return crypto.Equal(&pk, ar.GetPublicKey())
}

func (ar *ArbitratorImpl) IsOnDutyOfSide(sideChainKey string) bool {
	pk := crypto.PublicKey{}
	if err := pk.FromString(ArbitratorGroupSingleton.GetOnDutyArbitratorOfSide(sideChainKey)); err != nil {
		return false
	}
	return crypto.Equal(&pk, ar.GetPublicKey())
}

func (ar *ArbitratorImpl) GetArbitratorGroup() ArbitratorGroup {
	return ArbitratorGroupSingleton
}

func (ar *ArbitratorImpl) CreateWithdrawTransaction(
	withdrawInfoMap []*WithdrawInfo, sideChain SideChain, sideTransactionHash string, mcFunc MainChainFunc) []*tx.Transaction {

	var result []*tx.Transaction
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

func (ar *ArbitratorImpl) ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error) {
	return ar.mainChainImpl.ParseUserDepositTransactionInfo(txn)
}

func (ar *ArbitratorImpl) CreateDepositTransactions(proof spv.Proof, infoArray []*DepositInfo) map[*TransactionInfo]SideChain {

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

func (ar *ArbitratorImpl) BroadcastWithdrawProposal(txns []*tx.Transaction) {
	for _, txn := range txns {
		err := ar.mainChainImpl.BroadcastWithdrawProposal(txn)
		if err != nil {
			log.Warn(err.Error())
		}
	}
}

func (ar *ArbitratorImpl) SendWithdrawTransaction(txn *tx.Transaction) (interface{}, error) {
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

func (ar *ArbitratorImpl) Type() utcore.TransactionType {
	return utcore.TransferCrossChainAsset
}

func (ar *ArbitratorImpl) Confirmed() bool {
	return true
}

func (ar *ArbitratorImpl) Notify(proof spv.Proof, spvtxn utcore.Transaction) {
	if !ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfMain() {
		return
	}

	buf := new(bytes.Buffer)
	spvtxn.Serialize(buf)
	txBytes := buf.Bytes()

	r := bytes.NewReader(txBytes)
	txn := new(tx.Transaction)
	txn.Deserialize(r)

	depositInfo, err := ar.ParseUserDepositTransactionInfo(txn)
	if err != nil {
		log.Error(err)
		return
	}

	transactionInfoMap := ar.CreateDepositTransactions(proof, depositInfo)
	ar.SendDepositTransactions(transactionInfoMap)
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

	ar.spvService = NewSPVService(binary.LittleEndian.Uint64(publicKeyBytes), config.Parameters.MainNode.SpvSeedList)
	for _, sideNode := range config.Parameters.SideNodeList {
		if err = ar.spvService.RegisterAccount(sideNode.GenesisBlockAddress); err != nil {
			return err
		}
	}
	ar.spvService.RegisterTransactionListener(ar)

	go func() {
		if err = ar.spvService.Start(); err != nil {
			log.Error("spvService start failed ：", err)
		}
	}()

	return nil
}

func (ar *ArbitratorImpl) convertToTransactionContent(txn *tx.Transaction) (string, error) {
	buf := new(bytes.Buffer)
	err := txn.Serialize(buf)
	if err != nil {
		return "", err
	}
	content := common.BytesToHexString(buf.Bytes())
	return content, nil
}
