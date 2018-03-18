package main

import (
	"fmt"
	"Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/arbitration/arbitratorgroup"
	"os"
)

func main() {

	fmt.Printf("Arbitrators count: %d", base.Parameters.MemberCount)

	currentArbitrator, err := arbitratorgroup.ArbitratorGroupSingleton.GetCurrentArbitrator()
	if err != nil {
		fmt.Println("[Error] " + err.Error())
		os.Exit(1)
	}

	if !currentArbitrator.IsOnDuty() {
		fmt.Println("[Error] Current arbitrator is not on duty!")
		os.Exit(1)
	}
}
