package cs

import (
	"bytes"
	"errors"
	"sync"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA.SPV/peer"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
	"github.com/elastos/Elastos.ELA.Utility/p2p/server"
)

var (
	P2PClientSingleton *P2PClientAdapter
	spvP2PClient       server.IServer
)

const (
	//len of message need to less than 12
	WithdrawCommand                = "withdraw"
	ComplainCommand                = "complain"
	GetLastArbiterUsedUtxoCommand  = "RQLastUtxo"
	SendLastArbiterUsedUtxoCommand = "SDLastUtxo"

	MessageStoreHeight        = 5
	OpenService               = 1 << 2
	EIP001Version      uint32 = 10001
)

type P2PClientAdapter struct {
	listeners []base.P2PClientListener

	messageHashes map[common.Uint256]uint32
	cacheLock     sync.Mutex
}

func InitP2PClient(arbitrator Arbitrator) error {

	P2PClientSingleton = &P2PClientAdapter{
		messageHashes: make(map[common.Uint256]uint32, 0),
	}

	// Initiate P2P server configuration
	serverCfg := server.NewDefaultConfig(
		config.Parameters.Magic,
		EIP001Version,
		OpenService,
		config.Parameters.DefaultPort,
		config.Parameters.SeedList,
		nil,
		nil,
		nil,
		P2PClientSingleton.MakeMessage,
		func() uint64 { return uint64(0) },
	)
	serverCfg.MaxPeers = config.Parameters.MaxConnections
	serverCfg.DisableListen = true
	serverCfg.DisableRelayTx = true

	var err error
	spvP2PClient, err = server.NewServer(serverCfg)
	if err != nil {
		return err
	}

	return nil
}

func (adapter *P2PClientAdapter) tryInit() {
	if adapter.listeners == nil {
		adapter.listeners = make([]base.P2PClientListener, 0)
	}
}

func (adapter *P2PClientAdapter) Start() {
	spvP2PClient.Start()
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
		if currentMainChainHeight > MessageStoreHeight && v < currentMainChainHeight-MessageStoreHeight {
			needToDeleteMessages = append(needToDeleteMessages, k)
		}
	}
	for _, msg := range needToDeleteMessages {
		delete(adapter.messageHashes, msg)
	}

	return false
}

func (adapter *P2PClientAdapter) Broadcast(msg p2p.Message) {
	spvP2PClient.BroadcastMessage(msg)
}

func (adapter *P2PClientAdapter) HandleMessage(peer *peer.Peer, msg p2p.Message) error {
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
