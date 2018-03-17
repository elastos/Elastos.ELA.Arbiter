package arbitration

import (
	"fmt"
	"Elastos.ELA.Arbiter/crypto"
	"Elastos.ELA.Arbiter/common"
)

func main() {

	// initialize
	var arbitratorGroup ArbitratorGroup
	currentArbitrator := arbitratorGroup.GetCurrentArbitrator()
	currentArbitrator.GetComplainSolving().AddListener(currentArbitrator)

	// 1. post a complain request on web front
	var userKey *crypto.PublicKey
	var transactionHash common.Uint256
	//send to current arbitrator

	// 2. current arbitrator
	solvingContent, err := currentArbitrator.GetComplainSolving().AcceptComplain(userKey, transactionHash)
	if err != nil {
		currentArbitrator.GetComplainSolving().BroadcastComplainSolving(solvingContent)
	}

	//logic in Arbitrator.OnComplainFeedback（received other arbitrator's feedback, and complete the collecting stage）
	status := currentArbitrator.GetComplainSolving().GetComplainStatus(userKey, transactionHash)
	if status == Done {
		fmt.Println("Complain has been solved.")
	} else if status == Rejected {
		fmt.Println("Complain has been rejected.")
	}
}