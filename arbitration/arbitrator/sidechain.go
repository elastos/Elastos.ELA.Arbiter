package arbitrator

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA/common"
)

type SideChain interface {
	base.AccountListener
	SideChainNode

	GetKey() string
	GetExchangeRate() (float64, error)

	GetExistDepositTransactions(txs []string) ([]string, error)
	GetWithdrawTransaction(txHash string) (*base.WithdrawTxInfo, error)
	GetIllegalDeositTransaction(txHash string, height uint32) (bool, error)
	CheckIllegalEvidence(evidence *base.SidechainIllegalDataInfo) (bool, error)
	CheckIllegalDepositTx(depositTxs []common.Uint256) (bool, error)
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain

	StartSideChainMining()
	CheckAndRemoveWithdrawTransactionsFromDB() error
}
