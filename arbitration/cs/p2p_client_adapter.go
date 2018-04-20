package cs

import (
	"errors"
	"fmt"

	"encoding/binary"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/common/config"
	"github.com/elastos/Elastos.ELA.Arbiter/common/log"
	spvI "github.com/elastos/Elastos.ELA.SPV/interface"
	spvnet "github.com/elastos/Elastos.ELA.SPV/net"
	"github.com/elastos/Elastos.ELA.SPV/sdk"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
)

var (
	P2PClientSingleton *P2PClientAdapter
)

const (
	WithdrawCommand = "withdraw"
	ComplainCommand = "complain"

	GeneralListenerCount = 4
)

type P2PClientListener interface {
	OnP2PReceived(peer *spvnet.Peer, msg p2p.Message) error
}

type P2PClientAdapter struct {
	p2pClient  spvI.P2PClient
	listeners  []P2PClientListener
	arbitrator Arbitrator
}

func InitP2PClient(arbitrator Arbitrator) error {

	magic := config.Parameters.Magic
	seedList := config.Parameters.SeedList

	client := spvI.NewP2PClient(magic, seedList)
	P2PClientSingleton = &P2PClientAdapter{
		p2pClient:  client,
		arbitrator: arbitrator,
	}

	client.InitLocalPeer(P2PClientSingleton.InitLocalPeer)
	client.SetMessageHandler(P2PClientSingleton)

	client.Start()
	return nil
}

func (adapter *P2PClientAdapter) tryInit() {
	if adapter.listeners == nil {
		adapter.listeners = make([]P2PClientListener, GeneralListenerCount)
	}
}

func (adapter *P2PClientAdapter) Start() {
	adapter.p2pClient.Start()
}

func (adapter *P2PClientAdapter) InitLocalPeer(peer *spvnet.Peer) {
	publicKey := adapter.arbitrator.GetPublicKey()
	publicKeyBytes, _ := publicKey.EncodePoint(true)
	clientId := binary.LittleEndian.Uint64(publicKeyBytes)
	port := config.Parameters.NodePort

	peer.SetID(clientId)
	peer.SetPort(port)
	peer.SetRelay(uint8(1))
}

func (adapter *P2PClientAdapter) AddListener(listener P2PClientListener) {
	adapter.tryInit()
	adapter.listeners = append(adapter.listeners, listener)
}

func (adapter *P2PClientAdapter) Broadcast(msg p2p.Message) {
	adapter.p2pClient.PeerManager().Broadcast(msg)
}

func (adapter *P2PClientAdapter) HandleMessage(peer *spvnet.Peer, msg p2p.Message) error {
	if adapter.listeners == nil {
		return nil
	}

	for _, listener := range adapter.listeners {
		if err := listener.OnP2PReceived(peer, msg); err != nil {
			log.Warn(err)
			continue
		}
	}

	return nil
}

func (adapter *P2PClientAdapter) MakeMessage(cmd string) (message p2p.Message, err error) {
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

func (adapter *P2PClientAdapter) OnHandshake(v *msg.Version) error {

	if v.Version < sdk.ProtocolVersion {
		return errors.New(fmt.Sprint("To support SPV protocol, peer version must greater than ", sdk.ProtocolVersion))
	}

	//if v.Services/ServiveSPV&1 == 0 {
	//	return errors.New("SPV service not enabled on connected peer")
	//}

	return nil
}

func (adapter *P2PClientAdapter) OnPeerEstablish(peer *spvnet.Peer) {
	//peer.Send(msg.NewFilterLoad(spv.chain.GetBloomFilter()))
}
