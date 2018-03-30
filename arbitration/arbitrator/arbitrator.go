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
	spvdb "github.com/elastos/Elastos.ELA.SPV/db"
	. "github.com/elastos/Elastos.ELA.SPV/interface"
)

type Arbitrator interface {
	MainChain
	MainChainClient
	SideChainManager

	GetPublicKey() *crypto.PublicKey
	GetComplainSolving() ComplainSolving

	Sign(content []byte) ([]byte, error)

	IsOnDuty() bool
	GetArbitratorGroup() ArbitratorGroup

	InitAccount() error
	StartSpvModule() error
}

type ArbitratorImpl struct {
	mainChainImpl        MainChain
	mainChainClientImpl  MainChainClient
	sideChainManagerImpl SideChainManager
	spvService           SPVService
	keystore             Keystore
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

func (ar *ArbitratorImpl) IsOnDuty() bool {
	pk := crypto.PublicKey{}
	pk.FromString(ArbitratorGroupSingleton.GetOnDutyArbitrator())
	return crypto.Equal(&pk, ar.GetPublicKey())
}

func (ar *ArbitratorImpl) GetArbitratorGroup() ArbitratorGroup {
	return ArbitratorGroupSingleton
}

func (ar *ArbitratorImpl) CreateWithdrawTransaction(withdrawBank string, target string, amount common.Fixed64) (*tx.Transaction, error) {
	return ar.mainChainImpl.CreateWithdrawTransaction(withdrawBank, target, amount)
}

func (ar *ArbitratorImpl) ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error) {
	return ar.mainChainImpl.ParseUserDepositTransactionInfo(txn)
}

func (ar *ArbitratorImpl) BroadcastWithdrawProposal(txn *tx.Transaction) error {
	return ar.mainChainImpl.BroadcastWithdrawProposal(txn)
}

func (ar *ArbitratorImpl) ReceiveProposalFeedback(content []byte) error {
	return ar.mainChainImpl.ReceiveProposalFeedback(content)
}

func (ar *ArbitratorImpl) OnTransactionConfirmed(proof spvdb.Proof, spvtxn spvtx.Transaction) {
	if !ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDuty() {
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

	for _, info := range depositInfo {
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
		sideChain.SendTransaction(txInfo)
	}
}

func (ar *ArbitratorImpl) SignProposal(uint256 common.Uint256) error {
	return ar.mainChainClientImpl.SignProposal(uint256)
}

func (ar *ArbitratorImpl) OnReceivedProposal(content []byte) error {
	return ar.mainChainClientImpl.OnReceivedProposal(content)
}

func (ar *ArbitratorImpl) Feedback(transactionHash common.Uint256) error {
	return ar.mainChainClientImpl.Feedback(transactionHash)
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

	ar.spvService, err = NewSPVService(binary.LittleEndian.Uint64(publicKeyBytes))
	if err != nil {
		return err
	}
	for _, sideNode := range config.Parameters.SideNodeList {
		if err = ar.spvService.RegisterAccount(sideNode.GenesisBlockAddress); err != nil {
			return err
		}
	}
	ar.spvService.OnTransactionConfirmed(ar.OnTransactionConfirmed)
	if err = ar.spvService.Start(); err != nil {
		return err
	}

	return nil
}
