package base

import (
	"github.com/elastos/Elastos.ELA/p2p"
	"github.com/elastos/Elastos.ELA/p2p/peer"
)

type P2PClientListener interface {
	OnP2PReceived(peer *peer.Peer, msg p2p.Message) error
}
