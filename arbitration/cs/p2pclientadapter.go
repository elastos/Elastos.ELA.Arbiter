package cs

import (
	"encoding/hex"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/dpos/dtime"
	"github.com/elastos/Elastos.ELA/dpos/p2p"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	elap2p "github.com/elastos/Elastos.ELA/p2p"
)

var P2PClientSingleton *arbitratorsNetwork

const (
	//len of message need to less than 12
	DistributeItemCommand = "disitem"
	SendSchnorrItemommand = "senditem"
)

type messageItem struct {
	ID      peer.PID
	Message elap2p.Message
}

type arbitratorsNetwork struct {
	mainchainListeners []base.MainchainMsgListener

	peersLock      sync.Mutex
	connectedPeers []peer.PID

	p2pServer    p2p.Server
	messageQueue chan *messageItem
	quit         chan bool
}

func (n *arbitratorsNetwork) AddMainchainListener(listener base.MainchainMsgListener) {
	n.mainchainListeners = append(n.mainchainListeners, listener)
}

func (n *arbitratorsNetwork) Start() {
	n.p2pServer.Start()

	currentHeight := store.DbCache.MainChainStore.CurrentHeight(store.QueryHeightCode)
	peers, err := rpc.GetActiveDposPeers(currentHeight)
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

func (n *arbitratorsNetwork) UpdatePeers(connectedPeers []peer.PID) {
	n.peersLock.Lock()
	n.connectedPeers = connectedPeers
	for _, pid := range connectedPeers {
		n.p2pServer.AddAddr(pid, config.Parameters.DPoSNetAddress)
	}
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
	case SendSchnorrItemommand:
		proposal, ok := m.(*SendSchnorrProposalMessage)
		if ok {
			for _, v := range n.mainchainListeners {
				v.OnSendSchnorrItemMsg(msgItem.ID, proposal.NonceHash)
			}
		}

	}
}

func (n *arbitratorsNetwork) getNonce(pid peer.PID) uint64 {
	return rand.Uint64()
}

func (n *arbitratorsNetwork) sign(data []byte) []byte {
	sign, _ := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().Sign(data)
	return sign
}

func (n *arbitratorsNetwork) DumpArbiterPeersInfo() []*p2p.PeerInfo {
	return n.p2pServer.DumpPeersInfo()
}

func InitP2PClient(pid peer.PID) error {
	var err error
	P2PClientSingleton, err = NewArbitratorsNetwork(pid)
	return err
}

func NewArbitratorsNetwork(pid peer.PID) (*arbitratorsNetwork, error) {
	network := &arbitratorsNetwork{
		mainchainListeners: make([]base.MainchainMsgListener, 0),
		connectedPeers:     make([]peer.PID, 0),
		messageQueue:       make(chan *messageItem, 10000), //todo config handle capacity though config file
		quit:               make(chan bool),
	}
	notifier := p2p.NewNotifier(p2p.NFNetStabled|p2p.NFBadNetwork, network.notifyFlag)

	server, err := p2p.NewServer(&p2p.Config{
		DataDir:          filepath.Join(config.DataPath, config.DataDir, config.ArbiterDir),
		PID:              pid,
		MagicNumber:      config.Parameters.Magic,
		MaxNodePerHost:   config.Parameters.MaxNodePerHost,
		DefaultPort:      config.Parameters.NodePort,
		TimeSource:       dtime.NewMedianTime(),
		Sign:             network.sign,
		PingNonce:        network.getNonce,
		PongNonce:        network.getNonce,
		MakeEmptyMessage: makeEmptyMessage,
		HandleMessage:    network.handleMessage,
		StateNotifier:    notifier,
	})
	if err != nil {
		return nil, err
	}

	for _, p := range config.Parameters.CRCCrossChainArbiters {
		id := peer.PID{}
		pk, err := hex.DecodeString(p)
		if err != nil {
			return nil, errors.New("invalid CRC public key in config")
		}
		copy(id[:], pk)
		server.AddAddr(id, config.Parameters.DPoSNetAddress)
	}

	network.p2pServer = server

	return network, nil
}

func makeEmptyMessage(cmd string) (message elap2p.Message, err error) {
	switch cmd {
	case DistributeItemCommand:
		message = &DistributedItemMessage{}
	case SendSchnorrItemommand:
		message = &SendSchnorrProposalMessage{}
	default:
		return nil, errors.New("received unsupported message, CMD " + cmd)
	}
	return message, nil
}
