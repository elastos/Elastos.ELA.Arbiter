package arbitrator

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/crypto"
	spvtx "SPVWallet/core/transaction"
	spvdb "SPVWallet/db"
	spvInterface "SPVWallet/interface"
	"bytes"
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
	getSPVService() spvInterface.SPVService
}

type ArbitratorImpl struct {
	mainChainImpl        MainChain
	mainChainClientImpl  MainChainClient
	sideChainManagerImpl SideChainManager
	spvService           spvInterface.SPVService
}

func (ar *ArbitratorImpl) GetPublicKey() *crypto.PublicKey {
	//todo get from spv service
	//return ar.spvService.GetPublicKey()
	return nil
}

func (ar *ArbitratorImpl) GetComplainSolving() ComplainSolving {
	return nil
}

func (ar *ArbitratorImpl) Sign(content []byte) ([]byte, error) {
	return nil, nil
}

func (ar *ArbitratorImpl) IsOnDuty() bool {
	pk := crypto.PublicKey{}
	pk.FromString(ArbitratorGroupSingleton.GetOnDutyArbitrator())
	return pk == *ar.GetPublicKey()
}

func (ar *ArbitratorImpl) GetArbitratorGroup() ArbitratorGroup {
	return ArbitratorGroupSingleton
}

func (ar *ArbitratorImpl) getSPVService() spvInterface.SPVService {
	return ar.spvService
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
