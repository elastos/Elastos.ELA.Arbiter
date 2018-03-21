package arbitration

import (
	"Elastos.ELA.Arbiter/arbitration/arbitrator"
	"Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/crypto"
)

func main() {

	// initialize
	var pkDestroy *crypto.PublicKey
	var arbitratorGroup arbitrator.ArbitratorGroup
	currentArbitrator, err := arbitratorGroup.GetCurrentArbitrator()
	var sideAccountMonitor base.AccountMonitor
	sideAccountMonitor.SetAccount(pkDestroy)
	sideAccountMonitor.AddListener(currentArbitrator)
	currentArbitrator.GetArbitrationNet().AddListener(currentArbitrator)

	//1. wallet
	//var walleta wallet.Wallet
	//var amount, fee *common.Fixed64
	//var strAddress_a, strAddressS string
	//tx3, err := walleta.CreateTransaction(strAddress_a, strAddressS, amount, fee)
	//if tx3 == nil || err == nil {
	//	return
	//}
	//sign tx3
	//var transactionContent string
	//rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("Data", transactionContent))

	//2. arbitrator side chain
	//logic in SideChain.OnUTXOChanged (found a withdraw transaction)
	var transactionHash common.Uint256
	var pkDestroryAddress string
	sideChain, ok := currentArbitrator.GetChain(pkDestroryAddress)
	if !ok {
		return
	}

	pkS := sideChain.GetKey()
	hashList, err := sideChain.ParseUserMainChainHash(transactionHash)
	if valid, err := sideChain.IsTransactionValid(transactionHash); !valid || err != nil {
		return
	}

	//3. arbitrator side chain
	for _, hash := range hashList {
		tx4, err := currentArbitrator.CreateWithdrawTransaction(pkS, hash)
		tx4Bytes, err := tx4.Serialize()
		if err != nil {
			currentArbitrator.GetArbitrationNet().Broadcast(tx4Bytes)
		}
	}

	//logic in Arbitrator.OnReceived (received other arbitrator's feedback, and complete the collecting stage)
	tx4.Deserialize(tx4Bytes)
	//var tx4SignedContent string
	//rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("Data", tx4SignedContent))
}
