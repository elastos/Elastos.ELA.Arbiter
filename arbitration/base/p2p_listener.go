package base

import (
	spvnet "github.com/elastos/Elastos.ELA.SPV/net"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
)

type P2PClientListener interface {
	OnP2PReceived(peer *spvnet.Peer, msg p2p.Message) error
}
