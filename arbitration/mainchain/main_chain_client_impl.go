package mainchain

import (
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"

	"github.com/elastos/Elastos.ELA/p2p"
	"github.com/elastos/Elastos.ELA/p2p/peer"
)

type MainChainClientImpl struct {
	*DistributedNodeClient
}

func (client *MainChainClientImpl) OnP2PReceived(peer *peer.Peer, msg p2p.Message) error {
	if msg.CMD() != client.P2pCommand {
		return nil
	}

	switch m := msg.(type) {
	case *SignMessage:
		return client.OnReceivedProposal(m.Content)
	}
	return nil
}
