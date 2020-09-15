package arbitrator

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
)

type SideChain interface {
	base.AccountListener
	SideChainNode

	GetKey() string
	GetExchangeRate() (float64, error)

	GetExistDepositTransactions(txs []string) ([]string, error)
	GetWithdrawTransaction(txHash string) (*base.WithdrawTxInfo, error)
	GetFailedDepositTransaction(txHash string) (bool, error)
	CheckIllegalEvidence(evidence *base.SidechainIllegalDataInfo) (bool, error)
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain

	StartSideChainMining()
	CheckAndRemoveWithdrawTransactionsFromDB() error
	CheckAndRemoveReturnDepositTransactionsFromDB() error
	OnReceivedRegisteredSideChain() error
}
