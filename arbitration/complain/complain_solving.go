package complain

import (
	"Elastos.ELA.Arbiter/common"
	"errors"
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

type ComplainListener interface {
	OnComplainFeedback([]byte)
}

type ComplainSolving interface {
	AcceptComplain(userAddress, genesisBlockHash string, transactionHash common.Uint256) ([]byte, error)
	BroadcastComplainSolving([]byte) error

	GetComplainStatus(transactionHash common.Uint256) uint

	AddListener(listener ComplainListener)
}

type ComplainSolvingImpl struct {
	listeners         []ComplainListener
	complains         map[common.Uint256]ComplainItem
	finishedComplains map[common.Uint256]bool
}

func (comp *ComplainSolvingImpl) AcceptComplain(userAddress, genesisBlockHash string, transaction common.Uint256) ([]byte, error) {
	if _, ok := comp.complains[transaction]; ok {
		return nil, errors.New("Transaction already in solving list.")
	}
	if _, ok := comp.finishedComplains[transaction]; ok {
		return nil, errors.New("Transaction already solved.")
	}

	item := &ComplainItem{
		UserAddress:      userAddress,
		GenesisBlockHash: genesisBlockHash,
		TransactionHash:  transaction,
		IsFromMainBlock:  false}
	if len(genesisBlockHash) == 0 {
		item.IsFromMainBlock = true
	}

	return item.Serialize()
}

func (comp *ComplainSolvingImpl) BroadcastComplainSolving([]byte) error {
	//todo call p2p module to broadcast to other arbitrators
	return nil
}

func (comp *ComplainSolvingImpl) GetComplainStatus(transactionHash common.Uint256) uint {
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
	var item ComplainItem
	item.Deserialize(message)
	if _, ok := comp.complains[item.TransactionHash]; !ok {
		return errors.New("Unknown transcation.")
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

func (comp *ComplainSolvingImpl) createSolvingTransaction(item ComplainItem) error {
	//todo create solving transaction similar to tx4
	return nil
}
