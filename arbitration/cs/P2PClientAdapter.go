package cs

import (
	. "Elastos.ELA.Arbiter/arbitration/arbitrator"
	"Elastos.ELA.Arbiter/common/config"
	"Elastos.ELA.Arbiter/common/log"
	spvI "SPVWallet/interface"
	"SPVWallet/p2p"
	"encoding/binary"
	"errors"
	"fmt"
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

func (adapter *P2PClientAdapter) makeMessage(cmd string) (message p2p.Message, err error) {
	switch cmd {
	case WithdrawCommand:
		message = &SignMessage{Command: WithdrawCommand}
	case ComplainCommand:
		message = &SignMessage{Command: ComplainCommand}
	default:
		return nil, errors.New("Received unsupported message, CMD " + cmd)
	}
	return message, nil
}

func (adapter *P2PClientAdapter) handleVersion(v *p2p.Version) error {

	if v.Version < p2p.ProtocolVersion {
		return errors.New(fmt.Sprint("To support SPV protocol, peer version must greater than ", p2p.ProtocolVersion))
	}

	//if v.Services/ServiveSPV&1 == 0 {
	//	return errors.New("SPV service not enabled on connected peer")
	//}

	return nil
}

func (adapter *P2PClientAdapter) peerConnected(peer *p2p.Peer) {
	//peer.Send(msg.NewFilterLoad(spv.chain.GetBloomFilter()))
}

func init() {
	publicKey := ArbitratorGroupSingleton.GetCurrentArbitrator().GetPublicKey()
	publicKeyBytes, _ := publicKey.EncodePoint(true)
	clientId := binary.LittleEndian.Uint64(publicKeyBytes)
	magic := config.Parameters.Magic
	port := config.Parameters.NodePort
	seedList := config.Parameters.SeedList

	var client spvI.P2PClient
	client = spvI.NewP2PClient(clientId, magic, port, seedList)
	client.Start()
	P2PClientSingleton = &P2PClientAdapter{p2pClient: client}

	client.HandleMessage(P2PClientSingleton.fireP2PReceived)
	client.MakeMessage(P2PClientSingleton.makeMessage)
	client.HandleVersion(P2PClientSingleton.handleVersion)
	client.PeerConnected(P2PClientSingleton.peerConnected)
}
