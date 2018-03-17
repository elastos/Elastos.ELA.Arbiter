package arbitration

import (
	"Elastos.ELA.Arbiter/crypto"
	"Elastos.ELA.Arbiter/common"
)

type SideChain interface {
	AccountListener

	GetKey() *crypto.PublicKey
	GetNode() SideChainNode
	CreateDepositTransaction(target *crypto.PublicKey, information *SpvInformation) *TransactionInfo

	IsTransactionValid(transactionHash *common.Uint256) (bool, error)

	parseUserMainPublicKey(transactionHash *common.Uint256) *crypto.PublicKey
}

type SideChainManager interface {

	Add(chain SideChain) error
	Remove(key *crypto.PublicKey) error

	GetChain(key *crypto.PublicKey) (SideChain, error)
	GetAllChains() ([]SideChain, error)
}