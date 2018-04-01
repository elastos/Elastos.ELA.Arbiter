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
	lastSyncTime        *int64
	timeoutLimit        int64 //millisecond
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

func (group *ArbitratorGroupImpl) syncFromMainNode() error {
	currentTime := time.Now().UnixNano()
	if group.lastSyncTime != nil && (currentTime-*group.lastSyncTime)*int64(time.Millisecond) < group.timeoutLimit {
		return nil
	}

	group.mux.Lock()
	defer group.mux.Unlock()

	height, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return err
	}

	if group.currentHeight != nil && height == *group.currentHeight {
		return nil
	}

	groupInfo, err := rpc.GetArbitratorGroupInfoByHeight(height)
	if err != nil {
		return err
	}
	group.arbitrators = groupInfo.Arbitrators
	group.onDutyArbitratorIndex = groupInfo.OnDutyArbitratorIndex

	if group.dutyChangedCallback != nil {
		var onDutyPk crypto.PublicKey
		onDutyPk.FromString(group.GetOnDutyArbitrator())
		group.dutyChangedCallback(crypto.Equal(&onDutyPk, group.currentArbitrator.GetPublicKey()))
	}

	*group.currentHeight = height
	group.lastSyncTime = &currentTime
	return nil
}

func (group *ArbitratorGroupImpl) RegisterDutyChangedCallback(callback func(bool)) {
	group.dutyChangedCallback = callback
}

func (group *ArbitratorGroupImpl) GetArbitratorsCount() int {
	group.syncFromMainNode()

	group.mux.Lock()
	group.mux.Unlock()
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
		timeoutLimit: 1000,
	}

	currentArbitrator := &ArbitratorImpl{}
	ArbitratorGroupSingleton.currentArbitrator = currentArbitrator

	spvLog.Init()
}
