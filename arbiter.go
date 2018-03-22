package main

import (
	"time"

	. "Elastos.ELA.Arbiter/arbitration/arbitrator"
	"Elastos.ELA.Arbiter/arbitration/sidechain"
	"Elastos.ELA.Arbiter/common/config"
	"Elastos.ELA.Arbiter/common/log"
	"Elastos.ELA.Arbiter/net/servers/httpjsonrpc"
	"Elastos.ELA.Arbiter/store"
)

func SetSideChainAccountMonitor(arbitrator Arbitrator) {
	dataStore, err := store.OpenDataStore()
	if err != nil {
		log.Error("Side chain monitor setup error: ", err)
	}
	monitor := sidechain.SideChainAccountMonitorImpl{DataStore: dataStore}

	for _, side := range arbitrator.GetAllChains() {
		monitor.AddListener(side)
	}

	for {
		monitor.SyncChainData()
		time.Sleep(time.Millisecond * config.Parameters.SidechainMoniterScanInterval)
	}
}

func main() {
	currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()

	go ArbitratorGroupSingleton.SyncLoop()
	go SetSideChainAccountMonitor(currentArbitrator)
	// Start Server
	go httpjsonrpc.StartRPCServer()

	select {}
}
