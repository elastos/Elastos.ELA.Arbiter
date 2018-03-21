package main

import (
	"fmt"
	"os"
	"time"

	"Elastos.ELA.Arbiter/arbitration/arbitrator"
	"Elastos.ELA.Arbiter/arbitration/sidechain"
	//"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/common/config"
	"Elastos.ELA.Arbiter/common/log"
	"Elastos.ELA.Arbiter/net/servers/httpjsonrpc"
	"Elastos.ELA.Arbiter/store"
)

func SetSideChainAccountMonitor(arbitrator arbitrator.Arbitrator) {
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

	// SPV module init
	// Set listen addr
	/*
		db, err := wallet.GetDatabase()
		if err != nil {
			fmt.Println("[Error] " + err.Error())
			os.Exit(1)
		}
			for _, node := range config.Parameters.SideNodeList {
				GenesisBlockAddressBytes, err := common.HexStringToBytes(node.GenesisBlockAddress)
				if err == nil {
					redeemScript := CreateCrossChainRedeemScript(GenesisBlockAddressBytes)
					programHash, _ := transaction.ToProgramHash(redeemScript)
					db.AddAddress(nil, nil)
				}
			}
			// TODO heropan Set OnUTXOChanged and OnBlockHeightChanged callback
	*/

	currentArbitrator, err := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
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
