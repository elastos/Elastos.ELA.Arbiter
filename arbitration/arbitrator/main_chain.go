package arbitrator

import (
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	. "github.com/elastos/Elastos.ELA.Arbiter/common"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
	. "github.com/elastos/Elastos.ELA.Arbiter/store"
)

type MainChain interface {
	CreateWithdrawTransaction(withdrawBank string, target string, amount Fixed64,
		sideChainTransactionHash string, mcFunc MainChainFunc) (*tx.Transaction, error)
	ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error)

	BroadcastWithdrawProposal(txn *tx.Transaction) error
	ReceiveProposalFeedback(content []byte) error
}

type MainChainClient interface {
	OnReceivedProposal(content []byte) error
}

type MainChainFunc interface {
	GetAvailableUtxos(withdrawBank string) ([]*AddressUTXO, error)
	GetMainNodeCurrentHeight() (uint32, error)
}
