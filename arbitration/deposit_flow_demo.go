package arbitration

import (
	"Elastos.ELA.Arbiter/rpc"
	"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/crypto"
)

func main() {

	// initialize
	var pkS *crypto.PublicKey
	var arbitratorGroup ArbitratorGroup
	currentArbitrator := arbitratorGroup.GetCurrentArbitrator()
	var mainAccountMonitor AccountMonitor
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
	var transactionContent string
	rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("Data", transactionContent))

	//2. arbitrator main chain
	//logic in MainChain.OnUTXOChanged（found a deposit transaction）
	var transactionHash *common.Uint256
	pka := currentArbitrator.parseUserSidePublicKey(transactionHash)
	pkS = currentArbitrator.parseSideChainKey(transactionHash)
	spvInformation := currentArbitrator.GenerateSpvInformation(transactionHash)
	if valid, err := currentArbitrator.IsValid(spvInformation); !valid || err != nil {
		return
	}

	//3. arbitrator side chain
	sideChain, err := currentArbitrator.GetChain(pkS)
	if err != nil {
		tx2 := sideChain.CreateDepositTransaction(pka, spvInformation)
		sideChain.GetNode().SendTransaction(tx2)
	}
}
