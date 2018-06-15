package wallet

import (
	"encoding/json"
	"os"

	"github.com/elastos/Elastos.ELA.Client/log"
	. "github.com/elastos/Elastos.ELA.Client/rpc"

	// "github.com/cheggaaa/pb"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA/core"
)

type DataSync interface {
	SyncChainData()
}

type DataSyncImpl struct {
	DataStore
	addresses []*Address
}

func GetDataSync(dataStore DataStore) DataSync {
	return &DataSyncImpl{
		DataStore: dataStore,
	}
}

func (sync *DataSyncImpl) SyncChainData() {
	// Get the addresses in this wallet
	sync.addresses, _ = sync.GetAddresses()

	var chainHeight uint32
	var currentHeight uint32
	var needSync bool

	for {
		chainHeight, currentHeight, needSync = sync.needSyncBlocks()
		if !needSync {
			break
		}
		// bar := pb.StartNew(int(chainHeight - currentHeight + 1))
		for currentHeight <= chainHeight {
			hash, err := GetBlockHash(currentHeight)
			if err != nil {
				log.Error("Get block hash failed at height:", currentHeight, "error:", err)
				os.Exit(1)
			}
			block, err := GetBlock(hash)
			if err != nil {
				log.Error("Get block failed at height:", currentHeight, "error:", err)
				os.Exit(1)
			}
			sync.processBlock(block)

			// Update wallet height
			currentHeight = sync.CurrentHeight(block.Height + 1)
			// bar.Increment()
		}
		// bar.Finish()
	}
}

func (sync *DataSyncImpl) needSyncBlocks() (uint32, uint32, bool) {

	chainHeight, err := GetChainHeight()
	if err != nil {
		return 0, 0, false
	}

	currentHeight := sync.CurrentHeight(QueryHeightCode)

	if currentHeight >= chainHeight+1 {
		return chainHeight, currentHeight, false
	}

	return chainHeight, currentHeight, true
}

func (sync *DataSyncImpl) containAddress(address string) (*Address, bool) {
	for _, addr := range sync.addresses {
		if addr.Address == address {
			return addr, true
		}
	}
	return nil, false
}

func (sync *DataSyncImpl) processBlock(block *BlockInfo) {
	// Add UTXO to wallet address from transaction outputs
	for _, txInfo := range block.Tx {
		data, err := json.Marshal(txInfo)
		if err != nil {
			log.Error("Resolve transaction info failed")
			os.Exit(1)
		}
		var tx TransactionInfo
		err = json.Unmarshal(data, &tx)
		if err != nil {
			log.Error("Resolve transaction info failed")
			os.Exit(1)
		}
		// Add UTXOs to wallet address from transaction outputs
		for index, output := range tx.Outputs {
			if addr, ok := sync.containAddress(output.Address); ok {
				// Create UTXO input from output
				txHashBytes, _ := HexStringToBytes(tx.Hash)
				referTxHash, _ := Uint256FromBytes(BytesReverse(txHashBytes))
				lockTime := output.OutputLock
				if tx.TxType == CoinBase {
					lockTime = block.Height + 100
				}
				amount, _ := StringToFixed64(output.Value)
				// Save UTXO input to data store
				addressUTXO := &UTXO{
					Op:       NewOutPoint(*referTxHash, uint16(index)),
					Amount:   amount,
					LockTime: lockTime,
				}
				sync.AddAddressUTXO(addr.ProgramHash, addressUTXO)
			}
		}

		// Delete UTXOs from wallet by transaction inputs
		for _, input := range tx.Inputs {
			txHashBytes, _ := HexStringToBytes(input.TxID)
			referTxID, _ := Uint256FromBytes(BytesReverse(txHashBytes))
			sync.DeleteUTXO(NewOutPoint(*referTxID, input.VOut))
		}
	}
}
