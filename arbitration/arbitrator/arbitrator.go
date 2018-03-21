package arbitrator

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/arbitration/net"
	side "Elastos.ELA.Arbiter/arbitration/sidechain"
	"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/crypto"
	"Elastos.ELA.Arbiter/store"
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

	Sign(password []byte, item ComplainItem) ([]byte, error)

	IsOnDuty() bool
	GetArbitratorGroup() ArbitratorGroup
}

type ArbitratorImpl struct {
	store.Keystore
	sideChains map[string]side.SideChain
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

func (ar *ArbitratorImpl) Sign(password []byte, item ComplainItem) ([]byte, error) {
	return ar.Keystore.Sign(password, item)
}

func (ar *ArbitratorImpl) IsOnDuty() bool {
	return true
}

func (ar *ArbitratorImpl) GetArbitratorGroup() ArbitratorGroup {
	return &ArbitratorGroupSingleton
}

func (ar *ArbitratorImpl) CreateWithdrawTransaction(withdrawBank string, target common.Uint168) (*TransactionInfo, error) {
	return nil, nil
}

func (ar *ArbitratorImpl) ParseUserSideChainHash(hash common.Uint256) (map[common.Uint168]common.Uint168, error) {
	return nil, nil
}

func (ar *ArbitratorImpl) IsValid(information *SpvInformation) (bool, error) {
	return false, nil
}

func (ar *ArbitratorImpl) GenerateSpvInformation(transaction common.Uint256) *SpvInformation {
	return nil
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
