package complain

import (
	"errors"
	"sync"

	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
)

const (
	None = iota
	Solving
	Rejected
	Done
)

var (
	ComplainSolver ComplainSolving
)

type ComplainSolvingImpl struct {
	mux               sync.Mutex
	listeners         []ComplainListener
	complains         map[common.Uint256]*ComplainItemImpl
	finishedComplains map[common.Uint256]bool
}

func (comp *ComplainSolvingImpl) AcceptComplain(userAddress, genesisBlockHash string, transaction common.Uint256) ([]byte, error) {
	comp.mux.Lock()
	defer comp.mux.Unlock()

	if _, ok := comp.complains[transaction]; ok {
		return nil, errors.New("Complaint already in solving list.")
	}
	if _, ok := comp.finishedComplains[transaction]; ok {
		return nil, errors.New("Complaint already solved.")
	}

	item := &ComplainItemImpl{
		UserAddress:      userAddress,
		GenesisBlockHash: genesisBlockHash,
		TransactionHash:  transaction,
		IsFromMainBlock:  false}
	if len(genesisBlockHash) == 0 {
		item.IsFromMainBlock = true
	}

	if !item.Verify() {
		return nil, errors.New("Invalid complaint.")
	}

	comp.complains[transaction] = item
	return item.Serialize()
}

func (comp *ComplainSolvingImpl) BroadcastComplainSolving([]byte) error {
	//todo call p2p module to broadcast to other arbitrators
	return nil
}

func (comp *ComplainSolvingImpl) GetComplainStatus(transactionHash common.Uint256) uint {
	comp.mux.Lock()
	defer comp.mux.Unlock()

	_, ok := comp.complains[transactionHash]
	if !ok {
		success, ok := comp.finishedComplains[transactionHash]
		if !ok {
			return None
		}
		if success {
			return Done
		} else {
			return Rejected
		}
	} else {
		return Solving
	}
}

func (comp *ComplainSolvingImpl) AddListener(listener ComplainListener) {
	comp.listeners = append(comp.listeners, listener)
}

//todo called by p2p module feedback callback
func (comp *ComplainSolvingImpl) fireComplainFeedback(message []byte) error {
	comp.mux.Lock()
	defer comp.mux.Unlock()

	item, err := comp.mergeComplainItem(message)
	if err != nil {
		return err
	}

	if item.Accepted() {
		if err := comp.createSolvingTransaction(item); err != nil {
			comp.finishedComplains[item.TransactionHash] = false
			return err
		}

		comp.finishedComplains[item.TransactionHash] = true
		delete(comp.complains, item.TransactionHash)
	} else {
		comp.complains[item.TransactionHash] = item
	}
	return nil
}

func (comp *ComplainSolvingImpl) mergeComplainItem(message []byte) (*ComplainItemImpl, error) {
	var item ComplainItemImpl
	item.Deserialize(message)

	if _, ok := comp.complains[item.TransactionHash]; !ok {
		return nil, errors.New("Unknown transcation.")
	}

	//todo merge to value of comp.complains[item.TransactionHash]

	return comp.complains[item.TransactionHash], nil
}

func (comp *ComplainSolvingImpl) createSolvingTransaction(item *ComplainItemImpl) error {
	//todo create solving transaction similar to tx4
	return nil
}
