package arbitrator

import (
	"Elastos.ELA.Arbiter/common/config"
	"Elastos.ELA.Arbiter/common/log"
	"SPVWallet/interface"
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"
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

	lastSyncTime *int64
	timeoutLimit int64 //millisecond
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
	//todo synchronize from main chain block info
	group.arbitrators = append(group.arbitrators, "")
	group.arbitrators = append(group.arbitrators, "")
	group.onDutyArbitratorIndex = 0

	group.lastSyncTime = &currentTime
	return nil
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

	// SPV module init
	var err error
	publicKey := currentArbitrator.GetPublicKey()
	publicKeyBytes, _ := publicKey.EncodePoint(true)
	currentArbitrator.spvService, err = _interface.NewSPVService(binary.LittleEndian.Uint64(publicKeyBytes))
	if err != nil {
		fmt.Println("[Error] " + err.Error())
		os.Exit(1)
	}
	for _, sideNode := range config.Parameters.SideNodeList {
		currentArbitrator.spvService.RegisterAccount(sideNode.GenesisBlockAddress)
	}
	currentArbitrator.spvService.OnTransactionConfirmed(currentArbitrator.OnTransactionConfirmed)
	currentArbitrator.spvService.Start()
}
