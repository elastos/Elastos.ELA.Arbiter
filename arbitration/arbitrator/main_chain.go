package arbitrator

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	. "Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
	spvtx "SPVWallet/core/transaction"
	spvdb "SPVWallet/db"
)

type MainChain interface {
	CreateWithdrawTransaction(withdrawBank string, target Uint168, amount Fixed64) (*tx.Transaction, error)
	ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error)

	OnTransactionConfirmed(proof spvdb.Proof, spvtxn spvtx.Transaction)

	BroadcastWithdrawProposal(txn *tx.Transaction) error
	ReceiveProposalFeedback(content []byte) error
}

type MainChainClient interface {
	SignProposal(transactionHash Uint256) error
	OnReceivedProposal(content []byte) error
	Feedback(transactionHash Uint256) error
}
