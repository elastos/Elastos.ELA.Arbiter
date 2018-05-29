package sidechain

import (
	"errors"
	"time"

	"fmt"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
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
}

func (monitor *SideChainAccountMonitorImpl) tryInit() {
	if monitor.accountListenerMap == nil {
		monitor.accountListenerMap = make(map[string]AccountListener)
	}
	if monitor.onDutyMap == nil {
		monitor.onDutyMap = make(map[string]bool)
	}
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
				monitor.processTransactions(transactions, sideNode.GenesisBlockAddress)

				// Update wallet height
				currentHeight = store.DbCache.CurrentSideHeight(sideNode.GenesisBlockAddress, transactions.Height+1)

				fmt.Print(">")
			}
			fmt.Print("\n")
		}

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
	monitor.mux.Unlock()
	if !ok {
		return errors.New("Do not exist listener.")
	}

	pk, err := PublicKeyFromString(arbitrator.ArbitratorGroupSingleton.GetOnDutyArbitratorOfSide(genesisBlockAddress))
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

	sc, ok := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(genesisBlockAddress)
	if ok {
		sc.SetTick(sc.GetTick() + 1)
	}

	if onDuty && sc.GetTick() >= 5 {
		sc.SetTick(0)
		listener.StartSidechainMining()
		log.Info("Start side chain mining")
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

func (monitor *SideChainAccountMonitorImpl) processTransactions(transactions *BlockTransactions, genesisAddress string) {
	for _, txn := range transactions.Transactions {
		for _, output := range txn.Outputs {
			if output.Address == DESTROY_ADDRESS {
				if ok, err := store.DbCache.HasSideChainTx(txn.Hash); err != nil || !ok {
					monitor.fireUTXOChanged(txn, genesisAddress)
				}
			}
		}
	}
}

func (monitor *SideChainAccountMonitorImpl) syncUsedUtxo(height uint32, genesisAddress string) {
	sc, ok := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(genesisAddress)
	if !ok {
		log.Warn("[syncUsedUtxo] Get side chain from genesis address failed")
		return
	}
	sc.SetLastUsedUtxoHeight(height)
}
