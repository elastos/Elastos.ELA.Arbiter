package main

import (
	"os"
	"time"

	. "Elastos.ELA.Arbiter/arbitration/arbitrator"
	"Elastos.ELA.Arbiter/arbitration/cs"
	"Elastos.ELA.Arbiter/arbitration/mainchain"
	"Elastos.ELA.Arbiter/arbitration/sidechain"
	"Elastos.ELA.Arbiter/common/config"
	"Elastos.ELA.Arbiter/common/log"
	"Elastos.ELA.Arbiter/net/servers/httpjsonrpc"
	"Elastos.ELA.Arbiter/store"
)

func init() {
	log.Init(log.Path, log.Stdout)
}

func setSideChainAccountMonitor(arbitrator Arbitrator) {
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
		time.Sleep(time.Millisecond * config.Parameters.SideChainMonitorScanInterval)
	}
}

func initP2P(arbitrator Arbitrator) error {
	if err := cs.InitP2PClient(arbitrator); err != nil {
		return err
	}

	//register p2p client listener
	if err := mainchain.InitMainChain(arbitrator); err != nil {
		return err
	}

	cs.P2PClientSingleton.Start()
	return nil
}

func main() {
	log.Info("1. Init arbitrator configuration.")
	currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()

	log.Info("2. Init arbitrator account.")
	if err := currentArbitrator.InitAccount("123456"); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Info("3. Start arbitrator spv module.")
	if err := currentArbitrator.StartSpvModule(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Info("4. Start arbitrator P2P networks.")
	if err := initP2P(currentArbitrator); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Info("5. Start arbitrator group monitor.")
	go ArbitratorGroupSingleton.SyncLoop()

	log.Info("6. Start side chain account monitor.")
	go setSideChainAccountMonitor(currentArbitrator)
	// Start Server
	log.Info("7. Start servers.")
	go httpjsonrpc.StartRPCServer()

	select {}
}
