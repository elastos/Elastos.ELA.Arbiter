package arbitrator

import (
	"errors"
	"sync"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/crypto"
)

var (
	ArbitratorGroupSingleton *ArbitratorGroupImpl
)

type ArbitratorGroupListener interface {
	GetPublicKey() *crypto.PublicKey
	OnDutyArbitratorChanged(onDuty bool)
}

type ArbitratorGroup interface {
	GetCurrentArbitrator() Arbitrator
	GetArbitratorsCount() int
	GetAllArbitrators() []string
	GetOnDutyArbitratorOfMain() (string, error)
	CheckOnDutyStatus(uint32)
	SetListener(listener ArbitratorGroupListener)
}

type ArbitratorGroupImpl struct {
	mux sync.Mutex

	onDutyArbitratorIndex int
	arbitrators           []string
	currentArbitrator     Arbitrator

	currentHeight *uint32
	lastSyncTime  *uint64
	timeoutLimit  uint64 //millisecond

	listener         ArbitratorGroupListener
	isListenerOnDuty bool
}

func (group *ArbitratorGroupImpl) SyncLoop() {
	for {
		err := group.SyncFromMainNode()
		if err != nil {
			log.Error("Arbitrator group sync error: ", err)
		}

		time.Sleep(time.Millisecond * config.Parameters.SyncInterval)
	}
}

func (group *ArbitratorGroupImpl) InitArbitrators() error {
	return group.SyncFromMainNode()
}

func (group *ArbitratorGroupImpl) InitArbitratorsByStrings(arbiters []string, onDutyIndex int) {
	group.mux.Lock()
	defer group.mux.Unlock()

	group.arbitrators = arbiters
	group.onDutyArbitratorIndex = onDutyIndex
}

func (group *ArbitratorGroupImpl) SyncFromMainNode() error {
	currentTime := uint64(time.Now().UnixNano())
	if group.lastSyncTime != nil && (currentTime-*group.lastSyncTime)*uint64(time.Millisecond) < group.timeoutLimit {
		log.Info("[SyncFromMainNode] less than timeout limit")
		return nil
	}

	height, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		log.Info("[SyncFromMainNode] rpc get current height failed")
		return err
	}

	group.mux.Lock()
	if group.currentHeight != nil && *group.currentHeight != 0 && *group.currentHeight == height {
		group.mux.Unlock()
		return nil
	}
	group.mux.Unlock()

	var currentHeight uint32
	if mc := group.GetCurrentArbitrator().GetMainChain(); mc != nil {
		currentHeight = mc.SyncChainData()
	}
	groupInfo, err := rpc.GetArbitratorGroupInfoByHeight(currentHeight)
	if err != nil {
		log.Info("[SyncFromMainNode] get arbitrator group info failed")
		return err
	}

	group.mux.Lock()
	group.arbitrators = groupInfo.Arbitrators
	group.onDutyArbitratorIndex = groupInfo.OnDutyArbitratorIndex
	group.mux.Unlock()

	group.mux.Lock()
	*group.currentHeight = currentHeight
	group.lastSyncTime = &currentTime
	group.mux.Unlock()

	group.CheckOnDutyStatus(currentHeight)
	return nil
}

func (group *ArbitratorGroupImpl) CheckOnDutyStatus(height uint32) {
	if group.listener == nil {
		return
	}

	onDutyArbiter, err := ArbitratorGroupSingleton.GetOnDutyArbitratorOfMain()
	if err != nil {
		return
	}
	pk, err := base.PublicKeyFromString(onDutyArbiter)
	_, ok := group.listener.(*ArbitratorImpl)
	if ok && err == nil {
		if (group.isListenerOnDuty == false && crypto.Equal(group.listener.GetPublicKey(), pk)) ||
			(group.isListenerOnDuty == true && !crypto.Equal(group.listener.GetPublicKey(), pk)) {
			group.isListenerOnDuty = !group.isListenerOnDuty
			group.listener.OnDutyArbitratorChanged(group.isListenerOnDuty)
		} else if group.isListenerOnDuty == true && crypto.Equal(group.listener.GetPublicKey(), pk) && config.Parameters.CRClaimDPOSNodeStartHeight == height {
			group.listener.OnDutyArbitratorChanged(group.isListenerOnDuty)
		}
	} else if ok && err != nil {
		if group.isListenerOnDuty == true && pk == nil {
			group.isListenerOnDuty = !group.isListenerOnDuty
			group.listener.OnDutyArbitratorChanged(group.isListenerOnDuty)
		}
	}
}

func (group *ArbitratorGroupImpl) GetCurrentHeight() uint32 {
	group.mux.Lock()
	defer group.mux.Unlock()
	return *group.currentHeight
}

func (group *ArbitratorGroupImpl) GetArbitratorsCount() int {
	group.mux.Lock()
	defer group.mux.Unlock()
	return len(group.arbitrators)
}

func (group *ArbitratorGroupImpl) GetOnDutyArbitratorOfMain() (string, error) {
	group.mux.Lock()
	defer group.mux.Unlock()

	if len(group.arbitrators) == 0 || len(group.arbitrators) <= group.onDutyArbitratorIndex {
		return "", errors.New("Get arbitrators from main chain failed")
	}

	return group.arbitrators[group.onDutyArbitratorIndex], nil
}

func (group *ArbitratorGroupImpl) GetCurrentArbitrator() Arbitrator {
	group.mux.Lock()
	defer group.mux.Unlock()
	return group.currentArbitrator
}

func (group *ArbitratorGroupImpl) GetAllArbitrators() []string {
	group.mux.Lock()
	defer group.mux.Unlock()
	return group.arbitrators
}

func (group *ArbitratorGroupImpl) SetListener(listener ArbitratorGroupListener) {
	group.listener = listener
	group.isListenerOnDuty = false
}

func Init(client *account.Client) {
	ArbitratorGroupSingleton = &ArbitratorGroupImpl{
		timeoutLimit:     1000,
		currentHeight:    new(uint32),
		lastSyncTime:     new(uint64),
		isListenerOnDuty: false,
	}

	currentArbitrator := &ArbitratorImpl{mainOnDutyMux: new(sync.Mutex)}
	currentArbitrator.InitAccount(client)

	ArbitratorGroupSingleton.currentArbitrator = currentArbitrator
	ArbitratorGroupSingleton.SetListener(currentArbitrator)

	//spvLog.Init(config.Parameters.SPVPrintLevel, 20, 1024)
}
