package arbitrator

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
)

type MainChain interface {
	CreateWithdrawTransaction(sideChain SideChain, withdrawInfo *base.WithdrawInfo,
		sideChainTransactionHashes []string, mcFunc MainChainFunc) (*types.Transaction, error)

	BroadcastWithdrawProposal(txn *types.Transaction) error
	BroadcastSidechainIllegalData(data *base.SidechainIllegalData) error
	ReceiveProposalFeedback(content []byte) error

	SyncMainChainCachedTxs() error
	CheckAndRemoveDepositTransactionsFromDB() error
	SyncChainData()
}

type MainChainClient interface {
	OnReceivedProposal(id peer.PID, content []byte) error
}

type MainChainFunc interface {
	GetAvailableUtxos(withdrawBank string) ([]*store.AddressUTXO, error)
	GetMainNodeCurrentHeight() (uint32, error)
}
