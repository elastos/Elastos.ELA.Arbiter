package arbitrator

import (
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/bloom"
	"github.com/elastos/Elastos.ELA/core"
)

type SideChain interface {
	AccountListener
	SideChainNode

	GetKey() string
	GetRage() float32
	CreateDepositTransaction(target string, proof bloom.MerkleProof, amount common.Fixed64) (*TransactionInfo, error)
	ParseUserWithdrawTransactionInfo(txn *core.Transaction) ([]*WithdrawInfo, error)
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain
}
