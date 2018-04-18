package examples

import (
	"os"
	"testing"

	"bytes"
	"fmt"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/common/config"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
	"github.com/elastos/Elastos.ELA.Arbiter/crypto"
)

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func setup() {
	config.InitMockConfig()
}

//Collecting sign mostly used in withdraw procedure, which happens in the arbitrators' p2p network.
// Due to we can't collecting sign by order like normal multi-sign offline, the sign process is
// distributed and the current on duty arbitrator will extract the sign data and rearrange it.
func ExampleSignCollectingOfTwoArbitrators() {
	//We will simulate the collecting procedure without the p2p network transforming by letting
	// the two arbitrators within the same process.
	onDutyArbitrator := &arbitrator.ArbitratorImpl{}
	anotherArbitrator := &arbitrator.ArbitratorImpl{}

	//let's suppose we already have a withdraw transaction(like tx4 referenced in withdraw_procedure_test)
	var tx4 *tx.Transaction

	//--------------Part1(On duty arbitrator sending)-------------------------
	//note: the whole process happens on BroadcastWithdrawProposal method of DistributedNodeServer

	//step1.1 generate distributed item object for transforming and collecting
	programHash, _ := StandardAcccountPublicKeyToProgramHash(anotherArbitrator.GetPublicKey())
	transactionItem := &cs.DistributedItem{
		ItemContent:                 tx4,
		TargetArbitratorPublicKey:   anotherArbitrator.GetPublicKey(),
		TargetArbitratorProgramHash: programHash,
	}

	//step1.2 init redeem script for multi-sign(procedure of )
	publicKeys := make([]*crypto.PublicKey, 2)
	publicKeys[0] = onDutyArbitrator.GetPublicKey()
	publicKeys[1] = anotherArbitrator.GetPublicKey()
	redeemScript, _ := tx.CreateWithdrawRedeemScript(2, publicKeys)
	transactionItem.SetRedeemScript(redeemScript)

	//step1.3 on duty arbitrator sign she self
	transactionItem.Sign(onDutyArbitrator, false)

	//step1.4 serialize and send to another arbitrator
	buf := new(bytes.Buffer)
	transactionItem.Serialize(buf)
	proposal := buf.Bytes()

	//step1.5 init tx4 programs after serialize distributed item object
	tx4.Programs[0].Code = redeemScript
	tx4.Programs[0].Parameter = transactionItem.GetSignedData()

	//--------------Part2(Another arbitrator)-------------------------
	//note: the whole process happens on OnP2PReceived method of DistributedNodeClient

	//step2.1 deserialize the proposal
	transactionItem2 := &cs.DistributedItem{}
	transactionItem2.Deserialize(bytes.NewReader(proposal))

	//step2.2 another arbitrator sign the proposal
	transactionItem2.Sign(anotherArbitrator, true)

	//step2.3 reset item target and send back
	transactionItem2.TargetArbitratorPublicKey = onDutyArbitrator.GetPublicKey()
	programHash2, _ := StandardAcccountPublicKeyToProgramHash(onDutyArbitrator.GetPublicKey())
	transactionItem2.TargetArbitratorProgramHash = programHash2

	buf2 := new(bytes.Buffer)
	transactionItem2.Serialize(buf2)
	proposal2 := buf2.Bytes()

	//--------------Part2(On duty arbitrator receiving)-------------------------
	//note: the whole process happens on OnP2PReceived method of DistributedNodeServer

	//step3.1 deserialize the proposal
	transactionItem3 := &cs.DistributedItem{}
	transactionItem3.Deserialize(bytes.NewReader(proposal2))

	//step3.2 parse sign data of another arbitrator
	newSign, _ := transactionItem3.ParseFeedbackSignedData()

	//step3.3 merge new sign
	num, _ := MergeSignToTransaction(newSign, 1, tx4)

	//finally we have an valid transaction(tx4) that can be sent to main node

	fmt.Printf("Number of signature is: [%d]", num)
	//Output: 2
}
