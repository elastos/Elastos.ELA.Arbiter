package arbitrator

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
)

type MainChain interface {
	CreateWithdrawTransaction(withdrawBank string, target common.Uint168, amount common.Fixed64) (*TransactionInfo, error)
	ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error)

	BroadcastWithdrawProposal(content []byte) error
	ReceiveProposalFeedback(content []byte) error
}

type MainChainClient interface {
	SignProposal(password []byte, uint256 common.Uint256) error
	OnReceivedProposal(content []byte) error
	Feedback(transactionHash common.Uint256) error
}
