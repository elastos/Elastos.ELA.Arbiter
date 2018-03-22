package arbitrator

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
)

type MainChain interface {
	CreateWithdrawTransaction(withdrawBank string, target common.Uint168) (*TransactionInfo, error)
	ParseUserSideChainHash(hash common.Uint256) (map[common.Uint168]common.Uint168, error)

	BroadcastWithdrawProposal(content []byte) error
	ReceiveProposalFeedback(content []byte) error
}

type MainChainClient interface {
	SignProposal(password []byte, uint256 common.Uint256) error
	OnReceivedProposal(content []byte) error
	Feedback(transactionHash common.Uint256) error
}
