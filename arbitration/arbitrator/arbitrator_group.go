package arbitrator

import (
	"sync"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/common/config"
	"github.com/elastos/Elastos.ELA.Arbiter/common/log"
	"github.com/elastos/Elastos.ELA.Arbiter/crypto"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	spvLog "github.com/elastos/Elastos.ELA.SPV/spvwallet/log"
)

var (
	ArbitratorGroupSingleton *ArbitratorGroupImpl
	syncMainChain            bool
)

type ArbitratorsElection interface {
}

type ArbitratorGroup interface {
	ArbitratorsElection

	GetCurrentArbitrator() Arbitrator
	GetArbitratorsCount() int
	GetAllArbitrators() []string
	GetOnDutyArbitrator() string
}

type ArbitratorGroupImpl struct {
	mux sync.Mutex

	onDutyArbitratorIndex int
	arbitrators           []string
	currentArbitrator     Arbitrator

	currentHeight       *uint32
	dutyChangedCallback func(bool)
	lastSyncTime        *uint64
	timeoutLimit        uint64 //millisecond
}

func (group *ArbitratorGroupImpl) SyncLoop() {
	syncMainChain = false
	for {
		err := group.syncFromMainNode()
		if err != nil {
			log.Error("Arbitrator group sync error: ", err)
		}

		time.Sleep(time.Millisecond * config.Parameters.SyncInterval)
	}
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

	if group.dutyChangedCallback != nil && false == syncMainChain {
		var onDutyPk crypto.PublicKey
		syncMainChain = true
		onDutyPk.FromString(group.GetOnDutyArbitrator())
		syncMainChain = false
		group.dutyChangedCallback(crypto.Equal(&onDutyPk, group.currentArbitrator.GetPublicKey()))
	}

	//TODO add syncChainData [jzh]
	//group.currentArbitrator.SyncChainData()

	group.mux.Lock()
	*group.currentHeight = height
	group.lastSyncTime = &currentTime
	group.mux.Unlock()
	return nil
}

func (group *ArbitratorGroupImpl) RegisterDutyChangedCallback(callback func(bool)) {
	group.dutyChangedCallback = callback
}

func (group *ArbitratorGroupImpl) GetArbitratorsCount() int {
	group.syncFromMainNode()

	group.mux.Lock()
	defer group.mux.Unlock()
	return len(group.arbitrators)
}

func (group *ArbitratorGroupImpl) GetOnDutyArbitrator() string {
	group.syncFromMainNode()

	group.mux.Lock()
	defer group.mux.Unlock()
	return group.arbitrators[group.onDutyArbitratorIndex]
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

func init() {
	ArbitratorGroupSingleton = &ArbitratorGroupImpl{
		timeoutLimit:  1000,
		currentHeight: new(uint32),
		lastSyncTime:  new(uint64),
	}

	currentArbitrator := &ArbitratorImpl{}
	ArbitratorGroupSingleton.currentArbitrator = currentArbitrator

	spvLog.Init()
}
