package arbitrator

import (
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	. "github.com/elastos/Elastos.ELA.Arbiter/common"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
)

type MainChain interface {
	CreateWithdrawTransaction(withdrawBank string, target string, amount Fixed64) (*tx.Transaction, error)
	ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error)

	BroadcastWithdrawProposal(txn *tx.Transaction) error
	ReceiveProposalFeedback(content []byte) error
}

type MainChainClient interface {
	SignProposal(transactionHash Uint256) error
	OnReceivedProposal(content []byte) error
	Feedback(transactionHash Uint256) error
}
