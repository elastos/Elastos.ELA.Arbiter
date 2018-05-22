package arbitrator

import (
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA/bloom"
	"github.com/elastos/Elastos.ELA/core"
)

type SideChain interface {
	AccountListener
	P2PClientListener
	SideChainNode

	IsOnDuty() bool
	GetKey() string
	GetRage() float32

	CreateDepositTransaction(infoArray []*DepositInfo, proof bloom.MerkleProof,
		mainChainTransaction *core.Transaction) (*TransactionInfo, error)
	ParseUserWithdrawTransactionInfo(txn *core.Transaction) ([]*WithdrawInfo, error)
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain
}
