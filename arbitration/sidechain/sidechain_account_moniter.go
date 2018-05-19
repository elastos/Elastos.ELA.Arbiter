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
	"github.com/elastos/Elastos.ELA.Utility/crypto"
	"sync"
)

type SideChainAccountMonitorImpl struct {
	mux sync.Mutex

	ParentArbitrator   arbitrator.Arbitrator
	accountListenerMap map[string]AccountListener
	onDutyMap          map[string]bool
	tick               int
}

func (monitor *SideChainAccountMonitorImpl) tryInit() {
	if monitor.accountListenerMap == nil {
		monitor.accountListenerMap = make(map[string]AccountListener)
	}
	if monitor.onDutyMap == nil {
		monitor.onDutyMap = make(map[string]bool)
	}
	monitor.tick = 0
}

func (monitor *SideChainAccountMonitorImpl) AddListener(listener AccountListener) {
	monitor.tryInit()
	monitor.accountListenerMap[listener.GetAccountAddress()] = listener

	monitor.mux.Lock()
	monitor.onDutyMap[listener.GetAccountAddress()] = false
	monitor.mux.Unlock()
}

func (monitor *SideChainAccountMonitorImpl) RemoveListener(account string) error {
	if monitor.accountListenerMap == nil {
		return nil
	}

	if _, ok := monitor.accountListenerMap[account]; !ok {
		return errors.New("Do not exist listener.")
	}
	delete(monitor.accountListenerMap, account)
	delete(monitor.onDutyMap, account)
	return nil
}

func (monitor *SideChainAccountMonitorImpl) fireUTXOChanged(txinfo *TransactionInfo, genesisBlockAddress string) error {
	if monitor.accountListenerMap == nil {
		return nil
	}

	item, ok := monitor.accountListenerMap[genesisBlockAddress]
	if !ok {
		return errors.New("Fired unknown listener.")
	}

	return item.OnUTXOChanged(txinfo)
}

func (monitor *SideChainAccountMonitorImpl) SyncChainData(sideNode *config.SideNodeConfig) {
	for {
		chainHeight, currentHeight, needSync := monitor.needSyncBlocks(sideNode.GenesisBlockAddress, sideNode.Rpc)

		if needSync {
			for currentHeight < chainHeight {
				transactions, err := GetDestroyedTransactionByHeight(currentHeight, sideNode.Rpc)
				if err != nil {
					break
				}
				monitor.processTransactions(transactions)

				// Update wallet height
				currentHeight = store.DbCache.CurrentSideHeight(sideNode.GenesisBlockAddress, transactions.Height+1)

				fmt.Print(">")
			}
			fmt.Print("\n")
		}

		monitor.tick++
		monitor.checkOnDutyStatus(sideNode.GenesisBlockAddress)

		time.Sleep(time.Millisecond * config.Parameters.SideChainMonitorScanInterval)
	}
}

func (monitor *SideChainAccountMonitorImpl) checkOnDutyStatus(genesisBlockAddress string) error {
	if monitor.onDutyMap == nil || monitor.accountListenerMap == nil {
		return nil
	}

	listener, ok := monitor.accountListenerMap[genesisBlockAddress]
	monitor.mux.Lock()
	onDuty, _ := monitor.onDutyMap[genesisBlockAddress]
	tick := monitor.tick
	monitor.mux.Unlock()
	if !ok {
		return errors.New("Do not exist listener.")
	}

	pk, err := PublicKeyFromString(
		arbitrator.ArbitratorGroupSingleton.GetOnDutyArbitratorOfSide(genesisBlockAddress))
	if err != nil {
		return err
	}

	if (onDuty == true && !crypto.Equal(pk, monitor.ParentArbitrator.GetPublicKey())) ||
		(onDuty == false && crypto.Equal(pk, monitor.ParentArbitrator.GetPublicKey())) {
		monitor.mux.Lock()
		monitor.onDutyMap[genesisBlockAddress] = !onDuty
		monitor.mux.Unlock()

		listener.OnDutyArbitratorChanged(!onDuty)
	}

	if onDuty && tick == 5 {
		monitor.mux.Lock()
		monitor.tick = 0
		monitor.mux.Unlock()
		listener.StartSidechainMining()
	}
	return nil
}

func (monitor *SideChainAccountMonitorImpl) needSyncBlocks(genesisBlockAddress string, config *config.RpcConfig) (uint32, uint32, bool) {

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

func (monitor *SideChainAccountMonitorImpl) containDestroyAddress(address string) (string, bool) {
	for _, node := range config.Parameters.SideNodeList {
		if node.DestroyAddress == address {
			return node.GenesisBlockAddress, true
		}
	}
	return "", false
}

func (monitor *SideChainAccountMonitorImpl) processTransactions(transactions *BlockTransactions) {
	for _, txn := range transactions.Transactions {
		for _, output := range txn.Outputs {
			if genesisAddress, ok := monitor.containDestroyAddress(output.Address); ok {
				if ok, err := store.DbCache.HasSideChainTx(txn.Hash); err != nil || !ok {
					monitor.fireUTXOChanged(txn, genesisAddress)
				}
			}
		}
	}
}
