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
	spvtx "github.com/elastos/Elastos.ELA.SPV/core/transaction"
	. "github.com/elastos/Elastos.ELA.SPV/interface"
	spv "github.com/elastos/Elastos.ELA.SPV/interface"
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
	CreateWithdrawTransaction(withdrawBank string, target string,
		amount common.Fixed64, sideChainTransactionHash string) (*tx.Transaction, error)
	BroadcastWithdrawProposal(txn *tx.Transaction) error
}

type ArbitratorImpl struct {
	mainChainImpl        MainChain
	mainChainClientImpl  MainChainClient
	sideChainManagerImpl SideChainManager
	spvService           SPVService
	keystore             Keystore
}

func (ar *ArbitratorImpl) GetSideChainManager() SideChainManager {
	return ar.GetSideChainManager()
}

func (ar *ArbitratorImpl) GetPublicKey() *crypto.PublicKey {
	mainAccount := ar.keystore.MainAccount()

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
	mainAccount := ar.keystore.MainAccount()

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

func (ar *ArbitratorImpl) CreateWithdrawTransaction(withdrawBank string, target string,
	amount common.Fixed64, sideChainTransactionHash string) (*tx.Transaction, error) {
	return ar.mainChainImpl.CreateWithdrawTransaction(withdrawBank, target, amount, sideChainTransactionHash)
}

func (ar *ArbitratorImpl) ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error) {
	return ar.mainChainImpl.ParseUserDepositTransactionInfo(txn)
}

func (ar *ArbitratorImpl) CreateDepositTransactions(proof spv.Proof, infoArray []*DepositInfo) map[*TransactionInfo]SideChain {

	result := make(map[*TransactionInfo]SideChain, len(infoArray))
	for _, info := range infoArray {
		sideChain, ok := ar.GetChain(info.MainChainProgramHash.String())
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
		sideChain.SendTransaction(info)
	}
}

func (ar *ArbitratorImpl) BroadcastWithdrawProposal(txn *tx.Transaction) error {
	return ar.mainChainImpl.BroadcastWithdrawProposal(txn)
}

func (ar *ArbitratorImpl) ReceiveProposalFeedback(content []byte) error {
	return ar.mainChainImpl.ReceiveProposalFeedback(content)
}

func (ar *ArbitratorImpl) Type() spvtx.TransactionType {
	return spvtx.TransferCrossChainAsset
}

func (ar *ArbitratorImpl) Confirmed() bool {
	return true
}

func (ar *ArbitratorImpl) Notify(proof spv.Proof, spvtxn spvtx.Transaction) {
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

	ar.keystore = NewKeystore()
	_, err = ar.keystore.Open(string(passwd[:]))
	if err != nil {
		return err
	}
	accounts := ar.keystore.GetAccounts()
	if len(accounts) <= 0 {
		ar.keystore.NewAccount()
	}

	return nil
}

func (ar *ArbitratorImpl) StartSpvModule() error {
	publicKey := ar.keystore.MainAccount().PublicKey()
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
