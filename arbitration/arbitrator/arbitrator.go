package arbitrator

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/arbitration/net"
	"Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/crypto"
	spvTx "SPVWallet/core/transaction"
	spvInterface "SPVWallet/interface"
	spvMsg "SPVWallet/p2p/msg"
	"bytes"
)

type ArbitratorMain interface {
	MainChain
}

type ArbitratorSide interface {
	SideChainManager
}

type Arbitrator interface {
	ArbitratorMain
	ArbitratorSide
	net.ArbitrationNetListener
	ComplainListener

	GetPublicKey() *crypto.PublicKey
	GetArbitrationNet() net.ArbitrationNet
	GetComplainSolving() ComplainSolving

	Sign(content []byte) ([]byte, error)

	IsOnDuty() bool
	GetArbitratorGroup() ArbitratorGroup
	getSPVService() spvInterface.SPVService
}

type ArbitratorImpl struct {
	sideChains map[string]SideChain
	spvService spvInterface.SPVService
}

func (ar *ArbitratorImpl) GetPublicKey() *crypto.PublicKey {
	//todo get from spv service
	//return ar.spvService.GetPublicKey()
	return nil
}

func (ar *ArbitratorImpl) GetArbitrationNet() net.ArbitrationNet {
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
	return nil, nil
}

func (ar *ArbitratorImpl) ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error) {
	return nil, nil
}

func (ar *ArbitratorImpl) BroadcastWithdrawProposal(txn *tx.Transaction) error {
	return nil
}

func (ar *ArbitratorImpl) ReceiveProposalFeedback(content []byte) error {
	return nil
}

func (ar *ArbitratorImpl) CreateDepositTransaction(target common.Uint168, merkleBlock spvMsg.MerkleBlock, amount common.Fixed64) (*TransactionInfo, error) {
	return nil, nil
}

func (ar *ArbitratorImpl) IsTransactionValid(transactionHash common.Uint256) (bool, error) {
	return false, nil
}

func (ar *ArbitratorImpl) ParseUserWithdrawTransactionInfoParseUserWithdrawTransactionInfo(txn *tx.Transaction) ([]*WithdrawInfo, error) {
	return nil, nil
}

func (ar *ArbitratorImpl) OnTransactionConfirmed(merkleBlock spvMsg.MerkleBlock, trans []spvTx.Transaction) {
	for _, tran := range trans {
		buf := new(bytes.Buffer)
		tran.Serialize(buf)
		txBytes := buf.Bytes()

		r := bytes.NewReader(txBytes)
		txn := new(tx.Transaction)
		txn.Deserialize(r)
		depositInfo, err := ar.ParseUserDepositTransactionInfo(txn)
		if err != nil {
			//TODO heropan how to complain error
			continue
		}

		for _, info := range depositInfo {
			sideChain, ok := ar.GetChain(info.MainChainProgramHash.String())
			if !ok {
				//TODO heropan how to complain error
				continue
			}
			txInfo, err := sideChain.CreateDepositTransaction(info.TargetProgramHash, merkleBlock, info.Amount)
			if err != nil {
				//TODO heropan how to complain error
				continue
			}
			sideChain.GetNode().SendTransaction(txInfo)
		}
	}
}

func (ar *ArbitratorImpl) GetChain(key string) (SideChain, bool) {
	elem, ok := ar.sideChains[key]
	return elem, ok
}

func (ar *ArbitratorImpl) GetAllChains() []SideChain {
	var chains []SideChain
	for _, v := range ar.sideChains {
		chains = append(chains, v)
	}
	return chains
}

func (ar *ArbitratorImpl) OnReceived(buf []byte, arbitratorIndex int) {

}

func (ar *ArbitratorImpl) OnComplainFeedback([]byte) {

}
