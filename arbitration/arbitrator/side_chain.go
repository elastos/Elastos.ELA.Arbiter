package arbitrator

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
	spvMsg "SPVWallet/p2p/msg"
)

type SideChain interface {
	AccountListener
	SideChainNode

	GetKey() string
	CreateDepositTransaction(target common.Uint168, merkleBlock spvMsg.MerkleBlock, amount common.Fixed64) (*TransactionInfo, error)
	ParseUserWithdrawTransactionInfo(txn *tx.Transaction) ([]*WithdrawInfo, error)
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain
}
