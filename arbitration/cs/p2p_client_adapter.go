package cs

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/p2p"
	"github.com/elastos/Elastos.ELA/p2p/msg"
	"github.com/elastos/Elastos.ELA/p2p/peer"
	"github.com/elastos/Elastos.ELA/p2p/server"
)

var P2PClientSingleton *p2pclient

const (
	OpenService        = 1 << 2
	messageStoreHeight = 5
	defaultMaxPeers    = 12
)

const (
	//len of message need to less than 12
	WithdrawCommand                = "withdraw"
	ComplainCommand                = "complain"
	GetLastArbiterUsedUtxoCommand  = "RQLastUtxo"
	SendLastArbiterUsedUtxoCommand = "SDLastUtxo"
)

type p2pclient struct {
	server    server.IServer
	listeners []base.P2PClientListener

	cacheLock     sync.Mutex
	messageHashes map[common.Uint256]uint32

	newPeers  chan *peer.Peer
	donePeers chan *peer.Peer
	quit      chan struct{}
}

func InitP2PClient(dataDir string) error {
	maxPeers := config.Parameters.MaxConnections
	if maxPeers <= 0 {
		maxPeers = defaultMaxPeers
	}
	a := p2pclient{
		messageHashes: make(map[common.Uint256]uint32, 0),
		newPeers:      make(chan *peer.Peer, maxPeers),
		donePeers:     make(chan *peer.Peer, maxPeers),
	}
	// Initiate P2P server configuration
	serverCfg := server.NewDefaultConfig(
		config.Parameters.Magic,
		p2p.EIP001Version,
		OpenService,
		config.Parameters.NodePort,
		config.Parameters.SeedList,
		[]string{fmt.Sprint(":", config.Parameters.NodePort)},
		a.newPeer, a.donePeer,
		makeEmptyMessage,
		func() uint64 { return uint64(0) },
	)
	serverCfg.DataDir = dataDir
	serverCfg.MaxPeers = maxPeers
	log.Info("server config:", serverCfg)

	var err error
	a.server, err = server.NewServer(serverCfg)
	if err != nil {
		return err
	}

	P2PClientSingleton = &a

	return nil
}

func (c *p2pclient) newPeer(peer server.IPeer) {
	c.newPeers <- peer.ToPeer()
}

func (c *p2pclient) donePeer(peer server.IPeer) {
	c.donePeers <- peer.ToPeer()
}

// peerHandler handles new peers and done peers from P2P server.
// When comes new peer, create a spv peer warpper for it
func (c *p2pclient) peerHandler() {
	peers := make(map[*peer.Peer]struct{})

out:
	for {
		select {
		case p := <-c.newPeers:
			log.Debugf("p2pclient new peer %v", p)
			peers[p] = struct{}{}
			p.AddMessageFunc(c.handleMessage)

		case p := <-c.donePeers:
			_, ok := peers[p]
			if !ok {
				log.Errorf("unknown done peer %v", p)
				continue
			}

			delete(peers, p)
			log.Debugf("p2pclient done peer %v", p)

		case <-c.quit:
			break out
		}
	}

	// Drain any wait channels before we go away so we don't leave something
	// waiting for us.
cleanup:
	for {
		select {
		case <-c.newPeers:
		case <-c.donePeers:
		default:
			break cleanup
		}
	}
	log.Debug("Service peers handler done")
}

func (c *p2pclient) tryInit() {
	if c.listeners == nil {
		c.listeners = make([]base.P2PClientListener, 0)
	}
}

func (c *p2pclient) Start() {
	c.server.Start()
	go c.peerHandler()
}

func (c *p2pclient) Stop() {
	c.server.Stop()
	close(c.quit)
}

func (c *p2pclient) AddListener(listener base.P2PClientListener) {
	c.tryInit()
	c.listeners = append(c.listeners, listener)
}

func (c *p2pclient) GetMessageHash(msg p2p.Message) common.Uint256 {
	buf := new(bytes.Buffer)
	msg.Serialize(buf)
	msgHash := common.Sha256D(buf.Bytes())
	return msgHash
}

func (c *p2pclient) ExistMessageHash(msgHash common.Uint256) bool {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	for k := range c.messageHashes {
		if k == msgHash {
			return true
		}
	}
	return false
}

func (c *p2pclient) AddMessageHash(msgHash common.Uint256) bool {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	currentMainChainHeight := *ArbitratorGroupSingleton.GetCurrentHeight()
	c.messageHashes[msgHash] = currentMainChainHeight

	//delete message height 5 less than current main chain height
	var needToDeleteMessages []common.Uint256
	for k, v := range c.messageHashes {
		if currentMainChainHeight > messageStoreHeight && v < currentMainChainHeight-messageStoreHeight {
			needToDeleteMessages = append(needToDeleteMessages, k)
		}
	}
	for _, msg := range needToDeleteMessages {
		delete(c.messageHashes, msg)
	}

	return false
}

func (c *p2pclient) Broadcast(msg p2p.Message) {
	log.Debug("[Broadcast] msg:", msg.CMD())

	go func() {
		log.Debug("Broadcast peers", c.server.ConnectedPeers())
		c.server.BroadcastMessage(msg)
	}()

}

func (c *p2pclient) handleMessage(peer *peer.Peer, msg p2p.Message) {
	msgHash := c.GetMessageHash(msg)
	if c.ExistMessageHash(msgHash) {
		return
	} else {
		c.AddMessageHash(msgHash)
		log.Info("[HandleMessage] received msg:", msg.CMD(), "from peer id-", peer.ID())
		c.Broadcast(msg)
	}

	if c.listeners == nil {
		return
	}

	for _, listener := range c.listeners {
		if err := listener.OnP2PReceived(peer, msg); err != nil {
			log.Warn(err)
			continue
		}
	}

	return
}

func makeEmptyMessage(cmd string) (message p2p.Message, err error) {
	switch cmd {
	case p2p.CmdInv:
		message = new(msg.Inv)
	case p2p.CmdGetData:
		message = new(msg.GetData)
	case p2p.CmdNotFound:
		message = new(msg.NotFound)
	case p2p.CmdTx:
		message = msg.NewTx(new(types.Transaction))
	case p2p.CmdMerkleBlock:
		message = msg.NewMerkleBlock(new(types.Header))
	case p2p.CmdReject:
		message = new(msg.Reject)
	case WithdrawCommand:
		message = &SignMessage{Command: WithdrawCommand}
	case ComplainCommand:
		message = &SignMessage{Command: ComplainCommand}
	case GetLastArbiterUsedUtxoCommand:
		message = &GetLastArbiterUsedUTXOMessage{Command: GetLastArbiterUsedUtxoCommand}
	case SendLastArbiterUsedUtxoCommand:
		message = &SendLastArbiterUsedUTXOMessage{Command: SendLastArbiterUsedUtxoCommand}
	default:
		return nil, errors.New("Received unsupported message, CMD " + cmd)
	}
	return message, nil
}
