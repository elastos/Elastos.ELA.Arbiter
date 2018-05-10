package arbitrator

import (
	"sync"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	spvLog "github.com/elastos/Elastos.ELA.SPV/log"
)

var (
	ArbitratorGroupSingleton *ArbitratorGroupImpl
)

type ArbitratorsElection interface {
}

type ArbitratorGroup interface {
	ArbitratorsElection

	GetCurrentArbitrator() Arbitrator
	GetArbitratorsCount() int
	GetAllArbitrators() []string
	GetOnDutyArbitratorOfMain() string
	GetOnDutyArbitratorOfSide(sideChainKey string) string
}

type ArbitratorGroupImpl struct {
	mux sync.Mutex

	onDutyArbitratorIndex int
	arbitrators           []string
	currentArbitrator     Arbitrator

	currentHeight *uint32
	lastSyncTime  *uint64
	timeoutLimit  uint64 //millisecond
}

func (group *ArbitratorGroupImpl) SyncLoop() {
	for {
		err := group.syncFromMainNode()
		if err != nil {
			log.Error("Arbitrator group sync error: ", err)
		}

		time.Sleep(time.Millisecond * config.Parameters.SyncInterval)
	}
}

func (group *ArbitratorGroupImpl) InitArbitrators() error {
	return group.syncFromMainNode()
}

func (group *ArbitratorGroupImpl) InitArbitratorsByStrings(arbiters []string, onDutyIndex int) {
	group.mux.Lock()
	group.arbitrators = arbiters
	group.onDutyArbitratorIndex = onDutyIndex
	group.mux.Unlock()
}

func (group *ArbitratorGroupImpl) syncFromMainNode() error {
	currentTime := uint64(time.Now().UnixNano())
	if group.lastSyncTime != nil && currentTime*uint64(time.Millisecond) < group.timeoutLimit {
		return nil
	}
	height, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return err
	}

	group.mux.Lock()
	if group.currentHeight != nil && height == *group.currentHeight {
		group.mux.Unlock()
		return nil
	}
	group.mux.Unlock()

	groupInfo, err := rpc.GetArbitratorGroupInfoByHeight(height)
	if err != nil {
		return err
	}
	group.mux.Lock()
	group.arbitrators = groupInfo.Arbitrators
	group.onDutyArbitratorIndex = groupInfo.OnDutyArbitratorIndex
	group.mux.Unlock()

	//TODO add syncChainData [jzh]
	//group.currentArbitrator.SyncChainData()

	group.mux.Lock()
	*group.currentHeight = height
	group.lastSyncTime = &currentTime
	group.mux.Unlock()
	return nil
}

func (group *ArbitratorGroupImpl) GetArbitratorsCount() int {
	group.syncFromMainNode()

	group.mux.Lock()
	defer group.mux.Unlock()
	return len(group.arbitrators)
}

func (group *ArbitratorGroupImpl) GetOnDutyArbitratorOfMain() string {
	group.syncFromMainNode()

	group.mux.Lock()
	defer group.mux.Unlock()
	return group.arbitrators[group.onDutyArbitratorIndex]
}

func (group *ArbitratorGroupImpl) GetOnDutyArbitratorOfSide(sideChainKey string) string {

	height := store.DbCache.CurrentSideHeight(sideChainKey, store.QueryHeightCode)

	group.mux.Lock()
	defer group.mux.Unlock()
	index := int(height)
	index = index % len(group.arbitrators)
	return group.arbitrators[index]
}

func (group *ArbitratorGroupImpl) GetCurrentArbitrator() Arbitrator {
	group.syncFromMainNode()

	group.mux.Lock()
	defer group.mux.Unlock()
	return group.currentArbitrator
}

func (group *ArbitratorGroupImpl) GetAllArbitrators() []string {
	group.syncFromMainNode()

	group.mux.Lock()
	defer group.mux.Unlock()
	return group.arbitrators
}

func Init() {
	ArbitratorGroupSingleton = &ArbitratorGroupImpl{
		timeoutLimit:  1000,
		currentHeight: new(uint32),
		lastSyncTime:  new(uint64),
	}

	currentArbitrator := &ArbitratorImpl{mux: new(sync.Mutex)}
	ArbitratorGroupSingleton.currentArbitrator = currentArbitrator

	spvLog.Init()
}
