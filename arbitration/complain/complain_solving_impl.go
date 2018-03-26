package complain

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	. "Elastos.ELA.Arbiter/arbitration/cs"
	"Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
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
	*DistributedNodeServer
}

func (comp *ComplainSolvingImpl) AcceptComplain(userAddress, genesisBlockHash string, transaction common.Uint256) error {
	item := &ComplainItem{
		UserAddress:      userAddress,
		GenesisBlockHash: genesisBlockHash,
		TransactionHash:  transaction,
		IsFromMainBlock:  false}
	if len(genesisBlockHash) == 0 {
		item.IsFromMainBlock = true
	}

	trans, err := comp.CreateComplainTransaction(item)
	if err != nil {
		return err
	}

	return comp.BroadcastWithdrawProposal(trans)
}

func (comp *ComplainSolvingImpl) GetComplainStatus(transactionHash common.Uint256) uint {
	_, ok := comp.UnsolvedTransactions()[transactionHash]
	if !ok {
		success, ok := comp.FinishedTransactions()[transactionHash]
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

func (comp *ComplainSolvingImpl) CreateComplainTransaction(item *ComplainItem) (*tx.Transaction, error) {
	//todo append ComplainItem variables into attribute of transaction
	return nil, nil
}
