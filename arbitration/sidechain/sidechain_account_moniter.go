package sidechain

import (
	"fmt"

	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common/config"
	"Elastos.ELA.Arbiter/rpc"
	. "Elastos.ELA.Arbiter/store"
	"errors"
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

func (sync *SideChainAccountMonitorImpl) SyncChainData() {
	for _, node := range config.Parameters.SideNodeList {
		chainHeight, currentHeight, needSync := sync.needSyncBlocks(node.GenesisBlockAddress, node.Rpc)
		if !needSync {
			continue
		}

		for currentHeight < chainHeight {
			block, err := rpc.GetBlockByHeight(currentHeight, node.Rpc)
			if err != nil {
				break
			}
			sync.processBlock(block)

			// Update wallet height
			currentHeight = DbCache.CurrentSideHeight(node.GenesisBlockAddress, block.BlockData.Height+1)

			fmt.Print(">")
		}
	}

	fmt.Print("\n")
}

func (sync *SideChainAccountMonitorImpl) needSyncBlocks(genesisBlockAddress string, config *config.RpcConfig) (uint32, uint32, bool) {

	chainHeight, err := rpc.GetCurrentHeight(config)
	if err != nil {
		return 0, 0, false
	}

	currentHeight := DbCache.CurrentSideHeight(genesisBlockAddress, QueryHeightCode)

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

func (sync *SideChainAccountMonitorImpl) processBlock(block *BlockInfo) {
	// Add UTXO to wallet address from transaction outputs
	for _, txn := range block.Transactions {

		// Add UTXOs to wallet address from transaction outputs
		for _, output := range txn.Outputs {
			if genesisAddress, ok := sync.containDestroyAddress(output.Address); ok {
				sync.fireUTXOChanged(txn, genesisAddress)
			}
		}
	}
}
