package arbitrator

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
	spvdb "SPVWallet/db"
)

type SideChain interface {
	AccountListener
	SideChainNode

	GetKey() string
	CreateDepositTransaction(target common.Uint168, proof spvdb.Proof, amount common.Fixed64) (*TransactionInfo, error)
	ParseUserWithdrawTransactionInfo(txn *tx.Transaction) ([]*WithdrawInfo, error)
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain
}
