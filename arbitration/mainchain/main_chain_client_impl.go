package mainchain

import (
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	. "github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA.SPV/net"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
)

type MainChainClientImpl struct {
	*DistributedNodeClient
}

func (client *MainChainClientImpl) OnP2PReceived(peer *net.Peer, msg p2p.Message) error {
	if msg.CMD() != client.P2pCommand && msg.CMD() != WithdrawTxCacheClearCommand {
		return nil
	}

	switch m := msg.(type) {
	case *SignMessage:
		return client.OnReceivedProposal(m.Content)
	case *TxCacheClearMessage:
		err := DbCache.RemoveSideChainTxs(m.RemovedTxs)
		if err != nil {
			return err
		}
	}
	return nil
}
