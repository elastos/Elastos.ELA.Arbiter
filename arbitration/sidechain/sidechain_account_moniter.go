package sidechain

import (
	"errors"
	"time"

	"fmt"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	. "github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
)

type SideChainAccountMonitorImpl struct {
	accountListenerMap map[string]AccountListener
}

func (sync *SideChainAccountMonitorImpl) tryInit() {
	if sync.accountListenerMap == nil {
		sync.accountListenerMap = make(map[string]AccountListener)
	}
}

func (sync *SideChainAccountMonitorImpl) AddListener(listener AccountListener) {
	sync.tryInit()
	sync.accountListenerMap[listener.GetAccountAddress()] = listener
}

func (sync *SideChainAccountMonitorImpl) RemoveListener(account string) error {
	if sync.accountListenerMap == nil {
		return nil
	}

	if _, ok := sync.accountListenerMap[account]; !ok {
		return errors.New("Do not exist listener.")
	}
	delete(sync.accountListenerMap, account)
	return nil
}

func (sync *SideChainAccountMonitorImpl) fireUTXOChanged(txinfo *TransactionInfo, genesisBlockAddress string) error {
	if sync.accountListenerMap == nil {
		return nil
	}

	item, ok := sync.accountListenerMap[genesisBlockAddress]
	if !ok {
		return errors.New("Fired unknown listener.")
	}

	return item.OnUTXOChanged(txinfo)
}

func (sync *SideChainAccountMonitorImpl) SyncChainData(sideNode *config.SideNodeConfig) {

	for {
		chainHeight, currentHeight, needSync := sync.needSyncBlocks(sideNode.GenesisBlockAddress, sideNode.Rpc)

		if needSync {
			for currentHeight < chainHeight {
				transactions, err := GetDestroyedTransactionByHeight(currentHeight, sideNode.Rpc)
				if err != nil {
					break
				}
				sync.processTransactions(transactions)

				// Update wallet height
				currentHeight = store.DbCache.CurrentSideHeight(sideNode.GenesisBlockAddress, transactions.Height+1)

				fmt.Print(">")
			}
			fmt.Print("\n")
		}

		if arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfSide(sideNode.GenesisBlockAddress) {
			//add side chain mining related logic here
		}

		time.Sleep(time.Millisecond * config.Parameters.SideChainMonitorScanInterval)
	}
}

func (sync *SideChainAccountMonitorImpl) needSyncBlocks(genesisBlockAddress string, config *config.RpcConfig) (uint32, uint32, bool) {

	chainHeight, err := GetCurrentHeight(config)
	if err != nil {
		return 0, 0, false
	}

	currentHeight := store.DbCache.CurrentSideHeight(genesisBlockAddress, store.QueryHeightCode)

	if currentHeight >= chainHeight {
		return chainHeight, currentHeight, false
	}

	return chainHeight, currentHeight, true
}

func (sync *SideChainAccountMonitorImpl) containDestroyAddress(address string) (string, bool) {
	for _, node := range config.Parameters.SideNodeList {
		if node.DestroyAddress == address {
			return node.GenesisBlockAddress, true
		}
	}
	return "", false
}

func (sync *SideChainAccountMonitorImpl) processTransactions(transactions *BlockTransactions) {
	for _, txn := range transactions.Transactions {
		for _, output := range txn.Outputs {
			if genesisAddress, ok := sync.containDestroyAddress(output.Address); ok {
				if ok, err := store.DbCache.HashSideChainTx(txn.Hash); err != nil && !ok {
					sync.fireUTXOChanged(txn, genesisAddress)
				}
			}
		}
	}
}
