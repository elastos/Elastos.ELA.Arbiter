package main

import (
	"fmt"
	"os"

	"Elastos.ELA.Arbiter/arbitration/arbitratorgroup"
	"Elastos.ELA.Arbiter/arbitration/sidechain"
	//"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/common/config"
	"Elastos.ELA.Arbiter/net/servers/httpjsonrpc"
	//"SPVWallet/core"
	//"SPVWallet/core/transaction"
	//"SPVWallet/wallet"
)

func main() {

	fmt.Printf("Arbitrators count: %d", config.Parameters.MemberCount)

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

	currentArbitrator, err := arbitratorgroup.ArbitratorGroupSingleton.GetCurrentArbitrator()
	if err != nil {
		fmt.Println("[Error] " + err.Error())
		os.Exit(1)
	}

	if !currentArbitrator.IsOnDuty() {
		fmt.Println("[Error] Current arbitrator is not on duty!")
		os.Exit(1)
	}

	go sidechain.SetSidechainAccountMoniter()
	// Start Server
	go httpjsonrpc.StartRPCServer()

	select {}
}
