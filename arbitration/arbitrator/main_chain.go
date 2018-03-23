package arbitrator

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
	spvTx "SPVWallet/core/transaction"
	spvMsg "SPVWallet/p2p/msg"
)

type MainChain interface {
	CreateWithdrawTransaction(withdrawBank string, target common.Uint168, amount common.Fixed64) (*tx.Transaction, error)
	ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error)

	OnTransactionConfirmed(merkleBlock spvMsg.MerkleBlock, trans []spvTx.Transaction)

	BroadcastWithdrawProposal(txn *tx.Transaction) error
	ReceiveProposalFeedback(content []byte) error
}

type MainChainClient interface {
	SignProposal(uint256 common.Uint256) error
	OnReceivedProposal(content []byte) error
	Feedback(transactionHash common.Uint256) error
}
