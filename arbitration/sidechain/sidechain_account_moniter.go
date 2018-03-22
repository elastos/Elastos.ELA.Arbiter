package sidechain

import (
	"fmt"

	. "Elastos.ELA.Arbiter/arbitration/base"
	. "Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/common/config"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/rpc"
	"Elastos.ELA.Arbiter/store"
	"errors"
)

type SideChainAccountMonitorImpl struct {
	store.DataStore

	accountListenerMap map[string]AccountListener
}

func (sync *SideChainAccountMonitorImpl) AddListener(listener AccountListener) {
	sync.accountListenerMap[listener.GetAccountAddress()] = listener
}

func (sync *SideChainAccountMonitorImpl) RemoveListener(account string) error {
	if _, ok := sync.accountListenerMap[account]; !ok {
		return errors.New("Do not exist listener.")
	}
	delete(sync.accountListenerMap, account)
	return nil
}

func (sync *SideChainAccountMonitorImpl) fireUTXOChanged(txinfo *TransactionInfo, genesisBlockAddress string) error {
	item, ok := sync.accountListenerMap[genesisBlockAddress]
	if !ok {
		return errors.New("Fired unknown listener.")
	}

	return item.OnUTXOChanged(txinfo)
}

func (sync *SideChainAccountMonitorImpl) SyncChainData() {
	var chainHeight uint32
	var currentHeight uint32
	var needSync bool

	for _, node := range config.Parameters.SideNodeList {
		chainHeight, currentHeight, needSync = sync.needSyncBlocks(node.GenesisBlockAddress, node.Rpc)
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
			currentHeight = sync.CurrentHeight(node.GenesisBlockAddress, block.BlockData.Height+1)

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

	currentHeight := sync.CurrentHeight(genesisBlockAddress, store.QueryHeightCode)

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
		for index, output := range txn.Outputs {
			if genesisAddress, ok := sync.containDestroyAddress(output.Address); ok {
				// Create UTXO input from output
				txHashBytes, _ := HexStringToBytesReverse(txn.Hash)
				referTxHash, _ := Uint256FromBytes(txHashBytes)
				sequence := output.OutputLock
				if txn.TxType == tx.CoinBase {
					sequence = block.BlockData.Height + 100
				}
				input := &tx.UTXOTxInput{
					ReferTxID:          *referTxHash,
					ReferTxOutputIndex: uint16(index),
					Sequence:           sequence,
				}
				amount, _ := StringToFixed64(output.Value)
				// Save UTXO input to data store
				addressUTXO := &store.AddressUTXO{
					Input:               input,
					Amount:              amount,
					GenesisBlockAddress: genesisAddress,
					DestroyAddress:      output.Address,
				}
				sync.AddAddressUTXO(addressUTXO)

				sync.fireUTXOChanged(txn, genesisAddress)
			}
		}

		// Delete UTXOs from wallet by transaction inputs
		for _, input := range txn.UTXOInputs {
			txHashBytes, _ := HexStringToBytesReverse(input.ReferTxID)
			referTxID, _ := Uint256FromBytes(txHashBytes)
			txInput := &tx.UTXOTxInput{
				ReferTxID:          *referTxID,
				ReferTxOutputIndex: input.ReferTxOutputIndex,
				Sequence:           input.Sequence,
			}
			sync.DeleteUTXO(txInput)
		}
	}
}
