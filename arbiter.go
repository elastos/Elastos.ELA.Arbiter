package main

import (
	"fmt"
	"os"
	"time"

	"Elastos.ELA.Arbiter/arbitration/arbitratorgroup"
	"Elastos.ELA.Arbiter/arbitration/sidechain"
	"Elastos.ELA.Arbiter/common/config"
	"Elastos.ELA.Arbiter/common/log"
	"Elastos.ELA.Arbiter/net/servers/httpjsonrpc"
	"Elastos.ELA.Arbiter/store"
)

func SetSideChainAccountMonitor(arbitrator arbitratorgroup.Arbitrator) {
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

	fmt.Printf("Arbitrators count: %d \n", config.Parameters.MemberCount)

	currentArbitrator, err := arbitratorgroup.ArbitratorGroupSingleton.GetCurrentArbitrator()
	if err != nil {
		fmt.Println("[Error] " + err.Error())
		os.Exit(1)
	}

	if !currentArbitrator.IsOnDuty() {
		fmt.Println("[Error] Current arbitrator is not on duty!")
		os.Exit(1)
	}

	go SetSideChainAccountMonitor(currentArbitrator)
	// Start Server
	go httpjsonrpc.StartRPCServer()

	select {}
}
