package complain

import (
	"errors"

	"Elastos.ELA.Arbiter/arbitration/arbitratorgroup"
	. "Elastos.ELA.Arbiter/common"
)

type ComplainSolvingNode interface {
	OnReceived(message []byte) error
	Sign(password []byte, transactionHash Uint256) error
	Feedback(transactionHash Uint256) error
}

type ComplainSolvingNodeImpl struct {
	unsolvedComplains map[Uint256]*ComplainItemImpl
}

//todo call by p2p module
func (comp *ComplainSolvingNodeImpl) OnReceived(message []byte) error {
	var item ComplainItemImpl
	if err := item.Deserialize(message); err != nil {
		return err
	}

	_, ok := comp.unsolvedComplains[item.GetTransactionHash()]
	if ok {
		return errors.New("Complaint alread exist.")
	}

	if !item.Verify() {
		return errors.New("Invalid complaint.")
	}

	comp.unsolvedComplains[item.GetTransactionHash()] = &item
	return nil
}

func (comp *ComplainSolvingNodeImpl) Feedback(transactionHash Uint256) error {
	item, ok := comp.unsolvedComplains[transactionHash]
	if !ok {
		return errors.New("Can not find complaint.")
	}

	message, err := item.Serialize()
	if err != nil {
		return errors.New("Send complaint failed.")
	}

	return comp.sendBack(message)
}

func (comp *ComplainSolvingNodeImpl) sendBack(message []byte) error {
	//todo send feedback by p2p module
	return nil
}

func (comp *ComplainSolvingNodeImpl) Sign(password []byte, transactionHash Uint256) error {
	item, ok := comp.unsolvedComplains[transactionHash]
	if !ok {
		return errors.New("Can not find complaint.")
	}
	arbitrator, err := arbitratorgroup.ArbitratorGroupSingleton.GetCurrentArbitrator()
	if err != nil {
		return err
	}
	return item.SignItem(password, arbitrator)
}
