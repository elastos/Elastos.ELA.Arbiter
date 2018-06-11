package cs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	spvI "github.com/elastos/Elastos.ELA.SPV/interface"
	spvnet "github.com/elastos/Elastos.ELA.SPV/net"
	"github.com/elastos/Elastos.ELA.SPV/sdk"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
)

var (
	P2PClientSingleton *P2PClientAdapter
)

const (
	//len of message need to less than 12
	WithdrawCommand                = "withdraw"
	ComplainCommand                = "complain"
	GetLastArbiterUsedUtxoCommand  = "RQLastUtxo"
	SendLastArbiterUsedUtxoCommand = "SDLastUtxo"

	MessageStoreHeight = 5
)

type P2PClientAdapter struct {
	p2pClient  spvI.P2PClient
	listeners  []base.P2PClientListener
	arbitrator Arbitrator

	messageHashes map[common.Uint256]uint32
	cacheLock     sync.Mutex
}

func InitP2PClient(arbitrator Arbitrator) error {

	magic := config.Parameters.Magic
	seedList := config.Parameters.SeedList

	client := spvI.NewP2PClient(magic, seedList, config.Parameters.MinOutbound, config.Parameters.MaxConnections)
	P2PClientSingleton = &P2PClientAdapter{
		p2pClient:     client,
		arbitrator:    arbitrator,
		messageHashes: make(map[common.Uint256]uint32, 0),
	}

	client.InitLocalPeer(P2PClientSingleton.InitLocalPeer)
	client.SetMessageHandler(P2PClientSingleton)

	client.Start()
	return nil
}

func (adapter *P2PClientAdapter) tryInit() {
	if adapter.listeners == nil {
		adapter.listeners = make([]base.P2PClientListener, 0)
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

	peer.SetVersion(uint32(10007))
	peer.SetServices(uint64(4))
	peer.SetID(clientId)
	peer.SetPort(port)
	peer.SetRelay(uint8(1))
}

func (adapter *P2PClientAdapter) AddListener(listener base.P2PClientListener) {
	adapter.tryInit()
	adapter.listeners = append(adapter.listeners, listener)
}

func (adapter *P2PClientAdapter) GetMessageHash(msg p2p.Message) common.Uint256 {
	buf := new(bytes.Buffer)
	msg.Serialize(buf)
	msgHash := common.Sha256D(buf.Bytes())
	return msgHash
}

func (adapter *P2PClientAdapter) ExistMessageHash(msgHash common.Uint256) bool {
	adapter.cacheLock.Lock()
	defer adapter.cacheLock.Unlock()
	for k, _ := range adapter.messageHashes {
		if k == msgHash {
			return true
		}
	}
	return false
}

func (adapter *P2PClientAdapter) AddMessageHash(msgHash common.Uint256) bool {
	adapter.cacheLock.Lock()
	defer adapter.cacheLock.Unlock()
	currentMainChainHeight := *ArbitratorGroupSingleton.GetCurrentHeight()
	adapter.messageHashes[msgHash] = currentMainChainHeight

	//delete message height 5 less than current main chain height
	var needToDeleteMessages []common.Uint256
	for k, v := range adapter.messageHashes {
		if v < currentMainChainHeight-MessageStoreHeight {
			needToDeleteMessages = append(needToDeleteMessages, k)
		}
	}
	for _, msg := range needToDeleteMessages {
		delete(adapter.messageHashes, msg)
	}

	return false
}

func (adapter *P2PClientAdapter) Broadcast(msg p2p.Message) {
	adapter.p2pClient.PeerManager().Broadcast(msg)
}

func (adapter *P2PClientAdapter) HandleMessage(peer *spvnet.Peer, msg p2p.Message) error {
	msgHash := adapter.GetMessageHash(msg)
	if adapter.ExistMessageHash(msgHash) {
		return nil
	} else {
		adapter.AddMessageHash(msgHash)
		log.Info("*****")
		log.Info(msg.CMD())
		log.Info("*****")
		adapter.Broadcast(msg)
	}

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
	case GetLastArbiterUsedUtxoCommand:
		message = &GetLastArbiterUsedUTXOMessage{Command: GetLastArbiterUsedUtxoCommand}
	case SendLastArbiterUsedUtxoCommand:
		message = &SendLastArbiterUsedUTXOMessage{Command: SendLastArbiterUsedUtxoCommand}
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
