package complain

import (
	"bytes"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/core"
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

func (comp *ComplainSolvingImpl) AcceptComplain(userAddress, genesisBlockHash string, transactionHash common.Uint256) ([]byte, error) {
	item := &ComplainItem{
		UserAddress:      userAddress,
		GenesisBlockHash: genesisBlockHash,
		TransactionHash:  transactionHash,
		IsFromMainBlock:  false}
	if len(genesisBlockHash) == 0 {
		item.IsFromMainBlock = true
	}

	trans, err := comp.CreateComplainTransaction(item)
	if err != nil {
		return nil, err
	}

	//return comp.BroadcastWithdrawProposal(trans)

	buf := new(bytes.Buffer)
	trans.Serialize(buf)
	return buf.Bytes(), err
}

func (comp *ComplainSolvingImpl) BroadcastComplainSolving([]byte) error {
	return nil
}

func (comp *ComplainSolvingImpl) GetComplainStatus(transactionHash common.Uint256) uint {
	txs, err := store.DbCache.SideChainStore.GetSideChainTxsFromHashes([]string{transactionHash.String()})
	if err == nil && len(txs) != 0 {
		return Solving
	}

	/*txs, _, err = store.DbCache.GetMainChainTxsFromHashes([]string{transactionHash.String()})
	if err == nil && len(txs) != 0 {
		return Solving
	}*/

	succeedList, _, err := store.FinishedTxsDbCache.GetDepositTxByHash(transactionHash.String())
	if err == nil && len(succeedList) != 0 {
		for _, succeed := range succeedList {
			if succeed {
				return Done
			}
		}
		return Rejected
	}

	succeed, _, err := store.FinishedTxsDbCache.GetWithdrawTxByHash(transactionHash.String())
	if err == nil {
		if succeed {
			return Done
		} else {
			return Rejected
		}
	}

	return None
}

func (comp *ComplainSolvingImpl) CreateComplainTransaction(item *ComplainItem) (*core.Transaction, error) {
	//todo append ComplainItem variables into attribute of transaction
	return nil, nil
}
