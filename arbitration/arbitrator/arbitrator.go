package arbitrator

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/arbitration/net"
	side "Elastos.ELA.Arbiter/arbitration/sidechain"
	"Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/crypto"
	"Elastos.ELA.Arbiter/store"
	spvTx "SPVWallet/core/transaction"
	. "SPVWallet/interface"
	"SPVWallet/p2p/msg"
	"bytes"
)

type ArbitratorMain interface {
	MainChain
}

type ArbitratorSide interface {
	side.SideChainManager
}

type Arbitrator interface {
	ArbitratorMain
	ArbitratorSide
	net.ArbitrationNetListener
	ComplainListener

	GetPublicKey() *crypto.PublicKey
	GetProgramHash() *common.Uint168
	GetArbitrationNet() net.ArbitrationNet
	GetComplainSolving() ComplainSolving

	Sign(password []byte, content []byte) ([]byte, error)

	IsOnDuty() bool
	GetArbitratorGroup() ArbitratorGroup
	getSPVService() SPVService
}

type ArbitratorImpl struct {
	store.Keystore
	sideChains map[string]side.SideChain
	spvService SPVService
}

func (ar *ArbitratorImpl) GetPublicKey() *crypto.PublicKey {
	return ar.Keystore.GetPublicKey()
}

func (ar *ArbitratorImpl) GetProgramHash() *common.Uint168 {
	return ar.Keystore.GetProgramHash()
}

func (ar *ArbitratorImpl) GetArbitrationNet() net.ArbitrationNet {
	return nil
}

func (ar *ArbitratorImpl) GetComplainSolving() ComplainSolving {
	return nil
}

func (ar *ArbitratorImpl) Sign(password []byte, content []byte) ([]byte, error) {
	return nil, nil
}

func (ar *ArbitratorImpl) IsOnDuty() bool {
	return true
}

func (ar *ArbitratorImpl) GetArbitratorGroup() ArbitratorGroup {
	return &ArbitratorGroupSingleton
}

func (ar *ArbitratorImpl) getSPVService() SPVService {
	return ar.spvService
}

func (ar *ArbitratorImpl) CreateWithdrawTransaction(withdrawBank string, target common.Uint168) (*TransactionInfo, error) {
	return nil, nil
}

func (ar *ArbitratorImpl) ParseUserSideChainHash(txn *tx.Transaction) (map[common.Uint168]common.Uint168, error) {
	return nil, nil
}

func (ar *ArbitratorImpl) BroadcastWithdrawProposal(content []byte) error {
	return nil
}

func (ar *ArbitratorImpl) ReceiveProposalFeedback(content []byte) error {
	return nil
}

func (ar *ArbitratorImpl) CreateDepositTransaction(target common.Uint168, merkleBlock msg.MerkleBlock, txn *tx.Transaction) (*TransactionInfo, error) {
	return nil, nil
}

func (sc *ArbitratorImpl) IsTransactionValid(transactionHash common.Uint256) (bool, error) {
	return false, nil
}

func (sc *ArbitratorImpl) ParseUserMainChainHash(txn *tx.Transaction) ([]common.Uint168, error) {
	return nil, nil
}

func (ar *ArbitratorImpl) OnTransactionConfirmed(merkleBlock msg.MerkleBlock, trans []spvTx.Transaction) {
	for _, tran := range trans {
		buf := new(bytes.Buffer)
		tran.Serialize(buf)
		txBytes := buf.Bytes()

		r := bytes.NewReader(txBytes)
		txn := new(tx.Transaction)
		txn.Deserialize(r)
		hashMap, err := ar.ParseUserSideChainHash(txn)
		if err != nil {
			//TODO heropan how to complain error
			continue
		}

		for hashTarget, hashGenesis := range hashMap {
			sideChain, ok := ar.GetChain(hashGenesis.String())
			if !ok {
				//TODO heropan how to complain error
				continue
			}
			txInfo, err := sideChain.CreateDepositTransaction(hashTarget, merkleBlock, txn)
			if err == nil {
				//TODO heropan how to complain error
				sideChain.GetNode().SendTransaction(txInfo)
			}
		}
	}
}

func (ar *ArbitratorImpl) GetChain(key string) (side.SideChain, bool) {
	elem, ok := ar.sideChains[key]
	return elem, ok
}

func (ar *ArbitratorImpl) GetAllChains() []side.SideChain {
	var chains []side.SideChain
	for _, v := range ar.sideChains {
		chains = append(chains, v)
	}
	return chains
}

func (ar *ArbitratorImpl) OnReceived(buf []byte, arbitratorIndex int) {

}

func (ar *ArbitratorImpl) OnComplainFeedback([]byte) {

}
