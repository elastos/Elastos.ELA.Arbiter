package cs

import (
	"bytes"
	"errors"
	"math/rand"
	"os"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"

	"github.com/elastos/Elastos.ELA/dpos/p2p"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	elap2p "github.com/elastos/Elastos.ELA/p2p"
)

var P2PClientSingleton *arbitratorsNetwork

const (
	//len of message need to less than 12
	DistributeItemCommand          = "disitem"
	GetLastArbiterUsedUtxoCommand  = "RQLastUtxo"
	SendLastArbiterUsedUtxoCommand = "SDLastUtxo"
)

type messageItem struct {
	ID      peer.PID
	Message elap2p.Message
}

type arbitratorsNetwork struct {
	mainchainListeners []base.MainchainMsgListener
	sidechainListeners []base.SidechainMsgListener

	peersLock      sync.Mutex
	connectedPeers []p2p.PeerAddr

	p2pServer    p2p.Server
	messageQueue chan *messageItem
	quit         chan bool
}

func (n *arbitratorsNetwork) AddMainchainListener(listener base.MainchainMsgListener) {
	n.mainchainListeners = append(n.mainchainListeners, listener)
}

func (n *arbitratorsNetwork) AddSidechainListener(listener base.SidechainMsgListener) {
	n.sidechainListeners = append(n.sidechainListeners, listener)
}

func (n *arbitratorsNetwork) Start() {
	n.p2pServer.Start()

	peers, err := rpc.GetActiveDposPeers()
	if err != nil {
		log.Error("Get active dpos peers error when start, details: ", err)
		os.Exit(1)
	}
	n.UpdatePeers(peers)

	go func() {
	out:
		for {
			select {
			case msgItem := <-n.messageQueue:
				n.processMessage(msgItem)
			case <-n.quit:
				break out
			}
		}
	}()
}

func (n *arbitratorsNetwork) Stop() error {
	n.quit <- true
	return n.p2pServer.Stop()
}

func (n *arbitratorsNetwork) SendMessageToPeer(id peer.PID, msg elap2p.Message) error {
	return n.p2pServer.SendMessageToPeer(id, msg)
}

func (n *arbitratorsNetwork) BroadcastMessage(msg elap2p.Message) {
	n.peersLock.Lock()
	log.Info("[BroadcastMessage] current connected peers:", len(n.connectedPeers))
	n.peersLock.Unlock()

	n.p2pServer.BroadcastMessage(msg)
}

func (n *arbitratorsNetwork) UpdatePeers(connectedPeers []p2p.PeerAddr) {
	n.peersLock.Lock()
	n.connectedPeers = connectedPeers
	n.peersLock.Unlock()

	n.p2pServer.ConnectPeers(n.connectedPeers)
}

func (n *arbitratorsNetwork) notifyFlag(flag p2p.NotifyFlag) {
}

func (n *arbitratorsNetwork) handleMessage(pid peer.PID, msg elap2p.Message) {
	n.messageQueue <- &messageItem{pid, msg}
}

func (n *arbitratorsNetwork) processMessage(msgItem *messageItem) {
	m := msgItem.Message
	switch m.CMD() {
	case DistributeItemCommand:
		withdraw, processed := m.(*DistributedItemMessage)
		if processed {
			for _, v := range n.mainchainListeners {
				v.OnReceivedSignMsg(msgItem.ID, withdraw.Content)
			}
		}
	case GetLastArbiterUsedUtxoCommand:
		getUtxo, processed := m.(*GetLastArbiterUsedUTXOMessage)
		if processed {
			content := new(bytes.Buffer)
			getUtxo.Serialize(content)
			for _, v := range n.sidechainListeners {
				v.OnGetLastArbiterUsedUTXOMessage(msgItem.ID, content.Bytes())
			}
		}
	case SendLastArbiterUsedUtxoCommand:
		sendUtxo, processed := m.(*SendLastArbiterUsedUTXOMessage)
		if processed {
			content := new(bytes.Buffer)
			sendUtxo.Serialize(content)
			for _, v := range n.sidechainListeners {
				v.OnSendLastArbiterUsedUTXOMessage(msgItem.ID, content.Bytes())
			}
		}
	}
}

func (n *arbitratorsNetwork) getHearBeatNonce(pid peer.PID) uint64 {
	return rand.Uint64()
}

func (n *arbitratorsNetwork) signNonce(nonce []byte) (signature [64]byte) {
	sign, err := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().Sign(nonce)
	if err != nil || len(signature) != 64 {
		return signature
	}

	copy(signature[:], sign)

	return signature
}

func InitP2PClient(pid peer.PID) error {
	var err error
	P2PClientSingleton, err = NewArbitratorsNetwork(pid)
	return err
}

func NewArbitratorsNetwork(pid peer.PID) (*arbitratorsNetwork, error) {
	network := &arbitratorsNetwork{
		mainchainListeners: make([]base.MainchainMsgListener, 0),
		sidechainListeners: make([]base.SidechainMsgListener, 0),
		connectedPeers:     make([]p2p.PeerAddr, 0),
		messageQueue:       make(chan *messageItem, 10000), //todo config handle capacity though config file
		quit:               make(chan bool),
	}
	notifier := p2p.NewNotifier(p2p.NFNetStabled|p2p.NFBadNetwork, network.notifyFlag)

	server, err := p2p.NewServer(&p2p.Config{
		PID:              pid,
		MagicNumber:      config.Parameters.Magic,
		ProtocolVersion:  config.Parameters.Version,
		Services:         0, //todo add to config if need any services
		DefaultPort:      config.Parameters.NodePort,
		MakeEmptyMessage: makeEmptyMessage,
		HandleMessage:    network.handleMessage,
		PingNonce:        network.getHearBeatNonce,
		PongNonce:        network.getHearBeatNonce,
		SignNonce:        network.signNonce,
		StateNotifier:    notifier,
	})
	if err != nil {
		return nil, err
	}

	network.p2pServer = server

	return network, nil
}

func makeEmptyMessage(cmd string) (message elap2p.Message, err error) {
	switch cmd {
	case DistributeItemCommand:
		message = &DistributedItemMessage{}
	case GetLastArbiterUsedUtxoCommand:
		message = &GetLastArbiterUsedUTXOMessage{Command: GetLastArbiterUsedUtxoCommand}
	case SendLastArbiterUsedUtxoCommand:
		message = &SendLastArbiterUsedUTXOMessage{Command: SendLastArbiterUsedUtxoCommand}
	default:
		return nil, errors.New("received unsupported message, CMD " + cmd)
	}
	return message, nil
}
