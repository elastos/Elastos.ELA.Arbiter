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

	GetKey() string
	GetRage() float32

	SetLastUsedUtxoHeight(height uint32)
	GetLastUsedUtxoHeight() uint32
	GetLastUsedOutPoints() []core.OutPoint
	AddLastUsedOutPoints(ops []core.OutPoint)
	RemoveLastUsedOutPoints(ops []core.OutPoint)

	GetExistDepositTransactions(txs []string) ([]string, error)
	CreateDepositTransaction(depositInfo *DepositInfo, proof bloom.MerkleProof,
		mainChainTransaction *core.Transaction) (*TransactionInfo, error)
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain
	StartSideChainMining()
	CheckAndRemoveWithdrawTransactionsFromDB() error
}
