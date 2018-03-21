package arbitration

import (
	"Elastos.ELA.Arbiter/arbitration/arbitratorgroup"
	"Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/crypto"
	"Elastos.ELA.Arbiter/rpc"
)

func main() {

	// initialize
	var pkS *crypto.PublicKey
	var arbitratorGroup arbitratorgroup.ArbitratorGroup
	currentArbitrator, err := arbitratorGroup.GetCurrentArbitrator()
	var mainAccountMonitor base.AccountMonitor
	mainAccountMonitor.SetAccount(pkS)
	mainAccountMonitor.AddListener(currentArbitrator)

	//1. wallet
	//var walletA wallet.Wallet
	//var amount, fee *common.Fixed64
	//var strAddressA, strAddressS string
	//tx1, err := walletA.CreateTransaction(strAddressA, strAddressS, amount, fee)
	//if tx1 == nil || err == nil {
	//	return
	//}
	//sign tx1
	//var transactionContent string
	//rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("Data", transactionContent))

	//2. arbitrator main chain
	//logic in MainChain.OnUTXOChanged（found a deposit transaction）
	var transactionHash common.Uint256
	hashMap, err := currentArbitrator.ParseUserSideChainHash(transactionHash)
	spvInformation := currentArbitrator.GenerateSpvInformation(transactionHash)
	if valid, err := currentArbitrator.IsValid(spvInformation); !valid || err != nil {
		return
	}

	//3. arbitrator side chain
	for pka, pkSAddress := range hashMap {
		sideChain, ok := currentArbitrator.GetChain(pkSAddress.String())
		if ok {
			tx2, err := sideChain.CreateDepositTransaction(pka, spvInformation)
			sideChain.GetNode().SendTransaction(tx2)
		}
	}
}
