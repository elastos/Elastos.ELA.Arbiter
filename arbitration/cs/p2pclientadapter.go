package cs

import (
	"bytes"
	"errors"
	"math/rand"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/blockchain/interfaces"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/dpos"
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

	currentHeight uint32
	directPeers   map[string]*dpos.PeerItem
	peersLock     sync.Mutex
	store         store.PeerStore

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

	n.updateProducersInfo()
	n.UpdatePeers(arbitrator.ArbitratorGroupSingleton.GetAllArbitrators())

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

//todo update peers when NewElection(should defined in arbitrators group) is fired
func (n *arbitratorsNetwork) UpdatePeers(arbitrators []string) error {
	log.Info("[UpdatePeers] arbitrators:", arbitrators)
	for _, v := range arbitrators {
		n.peersLock.Lock()
		ad, ok := n.directPeers[v]
		if !ok {
			log.Error("can not find arbitrator related connection information, arbitrator public key is: ", v)
			n.peersLock.Unlock()
			continue
		}
		ad.NeedConnect = true
		ad.Sequence += uint32(len(arbitrators))
		n.peersLock.Unlock()
	}
	n.saveDirectPeers()

	return nil
}

func (n *arbitratorsNetwork) SendMessageToPeer(id peer.PID, msg elap2p.Message) error {
	return n.p2pServer.SendMessageToPeer(id, msg)
}

func (n *arbitratorsNetwork) BroadcastMessage(msg elap2p.Message) {
	log.Info("[BroadcastMessage] current connected peers:", len(n.getValidPeers()))
	n.p2pServer.BroadcastMessage(msg)
}

func (n *arbitratorsNetwork) ChangeHeight(height uint32) error {
	if height < n.currentHeight {
		return errors.New("changing height lower than current height")
	}

	offset := height - n.currentHeight
	if offset == 0 {
		return nil
	}

	n.peersLock.Lock()
	for _, v := range n.directPeers {
		if v.Sequence < offset {
			v.NeedConnect = false
			v.Sequence = 0
			continue
		}

		v.Sequence -= offset
	}

	peers := n.getValidPeers()
	for i, p := range peers {
		log.Info(" peer[", i, "] addr:", p.Addr, " pid:", common.BytesToHexString(p.PID[:]))
	}

	n.p2pServer.ConnectPeers(peers)
	n.peersLock.Unlock()

	go n.updateProducersInfo()

	n.currentHeight = height
	return nil
}

func (n *arbitratorsNetwork) getProducersConnectionInfo() (result map[string]p2p.PeerAddr) {
	//todo get producer connection info from ELA rpc
	return result
}

func (n *arbitratorsNetwork) getValidPeers() (result []p2p.PeerAddr) {
	result = make([]p2p.PeerAddr, 0)
	for _, v := range n.directPeers {
		if v.NeedConnect {
			result = append(result, v.Address)
		}
	}
	return result
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

func (n *arbitratorsNetwork) saveDirectPeers() {
	var peers []*interfaces.DirectPeers
	for k, v := range n.directPeers {
		if !v.NeedConnect {
			continue
		}
		pk, err := common.HexStringToBytes(k)
		if err != nil {
			continue
		}
		peers = append(peers, &interfaces.DirectPeers{
			PublicKey: pk,
			Address:   v.Address.Addr,
			Sequence:  v.Sequence,
		})
	}
	n.store.SaveDirectPeers(peers)
}

func (n *arbitratorsNetwork) getHearBeatNonce(pid peer.PID) uint64 {
	return rand.Uint64()
}

func (n *arbitratorsNetwork) updateProducersInfo() {
	log.Info("[updateProducersInfo] start")
	defer log.Info("[updateProducersInfo] end")
	connectionInfoMap := n.getProducersConnectionInfo()

	n.peersLock.Lock()
	defer n.peersLock.Unlock()

	needDeletedPeers := make([]string, 0)
	for k := range n.directPeers {
		if _, ok := connectionInfoMap[k]; !ok {
			needDeletedPeers = append(needDeletedPeers, k)
		}
	}
	for _, v := range needDeletedPeers {
		delete(n.directPeers, v)
	}

	for k, v := range connectionInfoMap {
		log.Info("[updateProducersInfo] peer id:", v.PID, " addr:", v.Addr)
		if _, ok := n.directPeers[k]; !ok {
			n.directPeers[k] = &dpos.PeerItem{
				Address:     v,
				NeedConnect: false,
				Peer:        nil,
				Sequence:    0,
			}
		}
	}

	n.saveDirectPeers()
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
	peerStore := store.DbCache.PeerStore
	network := &arbitratorsNetwork{
		mainchainListeners: make([]base.MainchainMsgListener, 0),
		sidechainListeners: make([]base.SidechainMsgListener, 0),
		directPeers:        make(map[string]*dpos.PeerItem),
		messageQueue:       make(chan *messageItem, 10000), //todo config handle capacity though config file
		quit:               make(chan bool),
		store:              peerStore,
		currentHeight:      store.DbCache.UTXOStore.CurrentHeight(0),
	}

	if peers, err := peerStore.GetDirectPeers(); err == nil {
		for _, p := range peers {
			pid := peer.PID{}
			copy(pid[:], p.PublicKey)
			network.directPeers[common.BytesToHexString(p.PublicKey)] = &dpos.PeerItem{
				Address: p2p.PeerAddr{
					PID:  pid,
					Addr: p.Address,
				},
				NeedConnect: true,
				Peer:        nil,
				Sequence:    p.Sequence,
			}
		}
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
