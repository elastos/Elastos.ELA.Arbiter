package arbitrator

import (
	"bytes"
	"encoding/binary"

	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/common/config"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/crypto"
	spvtx "SPVWallet/core/transaction"
	spvdb "SPVWallet/db"
	. "SPVWallet/interface"
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

	InitAccount(password string) error
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
	return pk == *ar.GetPublicKey()
}

func (ar *ArbitratorImpl) GetArbitratorGroup() ArbitratorGroup {
	return ArbitratorGroupSingleton
}

func (ar *ArbitratorImpl) CreateWithdrawTransaction(withdrawBank string, target common.Uint168, amount common.Fixed64) (*tx.Transaction, error) {
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
	buf := new(bytes.Buffer)
	spvtxn.Serialize(buf)
	txBytes := buf.Bytes()

	r := bytes.NewReader(txBytes)
	txn := new(tx.Transaction)
	txn.Deserialize(r)
	depositInfo, err := ar.ParseUserDepositTransactionInfo(txn)
	if err != nil {
		//TODO heropan how to complain error
		return
	}

	for _, info := range depositInfo {
		sideChain, ok := ar.GetChain(info.MainChainProgramHash.String())
		if !ok {
			//TODO heropan how to complain error
			continue
		}
		txInfo, err := sideChain.CreateDepositTransaction(info.TargetProgramHash, proof, info.Amount)
		if err != nil {
			//TODO heropan how to complain error
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

func (ar *ArbitratorImpl) InitAccount(password string) error {
	ar.keystore = NewKeystore()
	_, err := ar.keystore.Open(password)
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
