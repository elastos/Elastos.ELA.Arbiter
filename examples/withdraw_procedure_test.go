package examples

import (
	"fmt"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/sidechain"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
)

//This example demonstrate normal procedure of deposit
//As we known, the entire procedure will involve main chain, side chain, client of main chain
//	and client of side chain. To simplify this, we suppose the others are running well, and
//	we already known the result of these procedures.
func ExampleNormalWithdraw() {
	//--------------Part1(On client of side chain)-------------------------
	//Step1.1 create transaction(tx3)
	//	./ela-cli wallet -t create --withdraw EZwPHEMQLNBpP2VStF3gRk8EVoMM2i3hda --amount 1 --fee 0.1

	//Step1.2 sign tx3
	//	./ela-cli wallet -t sign --hex

	//Step1.3 send tx3 to side chain

	//--------------Part2(On side chain)-------------------------
	//Step2.1 tx3 has been confirmed and packaged in a new block

	//--------------Part3(On arbiter)-------------------------
	//let's suppose we get the object of current on duty arbitrator
	arbitrator := arbitrator.ArbitratorImpl{}

	//step3.1 SideChainAccountMonitorImpl found tx3 and fire utxo changed event

	//step3.2 SideChainImpl (correspond to side chain on part2) is a listener of SideChainAccountMonitorImpl,
	// the method OnUTXOChanged will be called automatically

	//let's suppose we already have the SideChainImpl object(from SideChainManager of the arbiter)
	sidechainObj := &sidechain.SideChainImpl{}

	//let's suppose we already known the serialized string of tx3 to simulate the object passed
	// from parameter of OnUTXOChanged
	//var tx3Str string
	var tx3 *tx.Transaction

	//step3.3 parse withdraw info from tx3
	withdrawInfos, _ := sidechainObj.ParseUserWithdrawTransactionInfo(tx3)

	//step3.4 create withdraw transactions(tx4) for main chain
	tx4s := arbitrator.CreateWithdrawTransaction(withdrawInfos, sidechainObj, tx3.Hash().String())

	//step3.5 broadcast withdraw proposal(contains tx4) into p2p network of arbitrators for collecting sign
	// from other arbitrators
	// arbitrator.BroadcastWithdrawProposal(tx4s)

	// note: details of broadcasting and sign collecting will be demonstrated in p2p_sign_collecting_test

	//step3.6 collecting enough sign then send tx4 to main chain

	//let's suppose we already known the serialized string of tx4 which contains complete signs
	//var signedTx4Str string
	var signedTx4 *tx.Transaction
	//arbitrator.SendWithdrawTransaction(signedTx4)

	//--------------Part4(On main chain)-------------------------
	//step4.1 main chain node received tx4

	//step4.2 special verify of tx4

	//step4.3 tx4 has been confirmed and packaged in a new block

	fmt.Printf("Unsigned withdraw transaction: [%s]\n, multi-signed withdraw transaction: [%s]", *tx4s[0], *signedTx4)
	//Output:
}
