package main

import (
	"fmt"
	"os"

	"Elastos.ELA.Arbiter/arbitration/arbitratorgroup"
	"Elastos.ELA.Arbiter/common/config"
	"Elastos.ELA.Arbiter/net/servers/httpjsonrpc"
)

func main() {

	fmt.Printf("Arbitrators count: %d", config.Parameters.MemberCount)

	currentArbitrator, err := arbitratorgroup.ArbitratorGroupSingleton.GetCurrentArbitrator()
	if err != nil {
		fmt.Println("[Error] " + err.Error())
		os.Exit(1)
	}

	if !currentArbitrator.IsOnDuty() {
		fmt.Println("[Error] Current arbitrator is not on duty!")
		os.Exit(1)
	}

	// Start Server
	go httpjsonrpc.StartRPCServer()

	select {}
}
