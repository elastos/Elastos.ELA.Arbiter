package arbitrator

import (
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/common"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
	spvdb "github.com/elastos/Elastos.ELA.SPV/db"
)

type SideChain interface {
	AccountListener
	SideChainNode

	GetKey() string
	GetRage() float32
	CreateDepositTransaction(target string, proof spvdb.Proof, amount common.Fixed64) (*TransactionInfo, error)
	ParseUserWithdrawTransactionInfo(txn *tx.Transaction) ([]*WithdrawInfo, error)
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain
}
