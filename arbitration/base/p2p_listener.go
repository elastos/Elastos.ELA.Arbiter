package base

import (
	"github.com/elastos/Elastos.ELA.SPV/peer"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
)

type P2PClientListener interface {
	OnP2PReceived(peer *peer.Peer, msg p2p.Message) error
}
