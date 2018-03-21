package arbitrator

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
)

type MainChain interface {
	CreateWithdrawTransaction(withdrawBank string, target common.Uint168) (*TransactionInfo, error)
	ParseUserSideChainHash(hash common.Uint256) (map[common.Uint168]common.Uint168, error)
}
