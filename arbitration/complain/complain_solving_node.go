package complain

import (
	"bytes"
	"errors"

	. "Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "Elastos.ELA.Arbiter/arbitration/base"
	. "Elastos.ELA.Arbiter/common"
)

type ComplainSolvingNode interface {
	OnReceived(message []byte) error
	Sign(transactionHash Uint256) error
	Feedback(transactionHash Uint256) error
}

type ComplainSolvingNodeImpl struct {
	unsolvedComplains map[Uint256]*DistributedItem
}

//todo call by p2p module
func (comp *ComplainSolvingNodeImpl) OnReceived(message []byte) error {
	complainItem := &DistributedItem{}
	if err := complainItem.Deserialize(bytes.NewReader(message)); err != nil {
		return err
	}

	complain, ok := complainItem.ItemContent.(*ComplainItemImpl)
	if !ok {
		return errors.New("Unknown complain content.")
	}
	if _, ok := comp.unsolvedComplains[complain.TransactionHash]; ok {
		return errors.New("Complaint already exit.")
	}
	if err := complain.Verify(); err != nil {
		return err
	}

	comp.unsolvedComplains[complain.TransactionHash] = complainItem

	if err := comp.Sign(complain.TransactionHash); err != nil {
		return err
	}

	if err := comp.Feedback(complain.TransactionHash); err != nil {
		return err
	}
	return nil
}

func (comp *ComplainSolvingNodeImpl) Feedback(transactionHash Uint256) error {
	item, ok := comp.unsolvedComplains[transactionHash]
	if !ok {
		return errors.New("Can not find complaint.")
	}

	ar := ArbitratorGroupSingleton.GetCurrentArbitrator()
	item.TargetArbitratorPublicKey = ar.GetPublicKey()

	programHash, err := StandardAcccountPublicKeyToProgramHash(item.TargetArbitratorPublicKey)
	if err != nil {
		return err
	}
	item.TargetArbitratorProgramHash = programHash

	messageReader := new(bytes.Buffer)
	err = item.Serialize(messageReader)
	if err != nil {
		return errors.New("Send complaint failed.")
	}

	return comp.sendBack(messageReader.Bytes())
}

func (comp *ComplainSolvingNodeImpl) Sign(transactionHash Uint256) error {
	item, ok := comp.unsolvedComplains[transactionHash]
	if !ok {
		return errors.New("Can not find complaint.")
	}
	return item.Sign(ArbitratorGroupSingleton.GetCurrentArbitrator())
}

func (comp *ComplainSolvingNodeImpl) sendBack(message []byte) error {
	//todo send feedback by p2p module
	return nil
}
