package cs

import (
	spvI "SPVWallet/interface"
	"SPVWallet/p2p"
)

var (
	P2PClientSingleton *P2PClientAdapter
)

type P2PClientListener interface {
	OnP2PReceived(peer *p2p.Peer, msg p2p.Message)
}

type P2PClientAdapter struct {
	p2pClient spvI.P2PClient
	listeners []P2PClientListener
}

func (adapter *P2PClientAdapter) AddListener(listener P2PClientListener) {
	adapter.listeners = append(adapter.listeners, listener)
}

func (adapter *P2PClientAdapter) Broadcast(msg p2p.Message) {
	adapter.p2pClient.PeerManager().Broadcast(msg)
}

func init() {
	var client spvI.P2PClient
	//client = spvI.NewP2PClient()
	client.Start()
	P2PClientSingleton = &P2PClientAdapter{p2pClient: client}
}
