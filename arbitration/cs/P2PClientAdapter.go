package cs

import (
	"Elastos.ELA.Arbiter/common/log"
	spvI "SPVWallet/interface"
	"SPVWallet/p2p"
)

var (
	P2PClientSingleton *P2PClientAdapter
)

const (
	WithdrawCommand = "withdraw"
	ComplainCommand = "complain"
)

type P2PClientListener interface {
	OnP2PReceived(peer *p2p.Peer, msg p2p.Message) error
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

func (adapter *P2PClientAdapter) fireP2PReceived(peer *p2p.Peer, msg p2p.Message) error {
	for _, listener := range adapter.listeners {
		if err := listener.OnP2PReceived(peer, msg); err != nil {
			log.Warn(err)
			continue
		}
	}

	return nil
}

func init() {
	var client spvI.P2PClient
	//client = spvI.NewP2PClient()
	client.Start()
	P2PClientSingleton = &P2PClientAdapter{p2pClient: client}
	client.HandleMessage(P2PClientSingleton.fireP2PReceived)
}
