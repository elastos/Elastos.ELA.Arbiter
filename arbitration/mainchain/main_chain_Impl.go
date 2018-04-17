package mainchain

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	. "github.com/elastos/Elastos.ELA.Arbiter/common"
	"github.com/elastos/Elastos.ELA.Arbiter/common/config"
	pg "github.com/elastos/Elastos.ELA.Arbiter/core/program"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
	"github.com/elastos/Elastos.ELA.Arbiter/core/transaction/payload"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	. "github.com/elastos/Elastos.ELA.Arbiter/store"
	spvWallet "github.com/elastos/Elastos.ELA.SPV/spvwallet"
)

const WithdrawAssetLockTime uint32 = 6

type MainChainImpl struct {
	*DistributedNodeServer
}

func (mc *MainChainImpl) CreateWithdrawTransaction(withdrawBank string, target string, amount Fixed64,
	sideChainTransactionHash string) (*tx.Transaction, error) {

	mc.syncChainData()

	// Check if from address is valid
	assetID := spvWallet.SystemAssetId
	programhash, err := Uint168FromAddress(target)
	if err != nil {
		return nil, err
	}
	// Create transaction outputs
	var totalOutputAmount = amount
	var txOutputs []*tx.TxOutput
	txOutput := &tx.TxOutput{
		AssetID:     Uint256(assetID),
		ProgramHash: *programhash,
		Value:       amount,
		OutputLock:  uint32(WithdrawAssetLockTime),
	}

	txOutputs = append(txOutputs, txOutput)

	utxos, err := DbCache.GetAddressUTXOsFromGenesisBlockAddress(withdrawBank)
	if err != nil {
		return nil, errors.New("Get spender's UTXOs failed.")
	}
	availableUTXOs := mc.getAvailableUTXOs(utxos)
	availableUTXOs = SortUTXOs(availableUTXOs)

	// Create transaction inputs
	var txInputs []*tx.UTXOTxInput
	for _, utxo := range availableUTXOs {
		txInputs = append(txInputs, utxo.Input)
		if *utxo.Amount < totalOutputAmount {
			totalOutputAmount -= *utxo.Amount
		} else if *utxo.Amount == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if *utxo.Amount > totalOutputAmount {
			programHash, err := Uint168FromAddress(withdrawBank)
			if err != nil {
				return nil, err
			}
			change := &tx.TxOutput{
				AssetID:     Uint256(assetID),
				Value:       Fixed64(*utxo.Amount - totalOutputAmount),
				OutputLock:  uint32(0),
				ProgramHash: *programHash,
			}
			txOutputs = append(txOutputs, change)
			totalOutputAmount = 0
			break
		}
	}

	if totalOutputAmount > 0 {
		return nil, errors.New("Available token is not enough")
	}

	redeemScript, err := CreateRedeemScript()
	if err != nil {
		return nil, err
	}

	// Create payload
	chainHeight, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}

	txPayload := &payload.WithdrawAsset{
		BlockHeight:              chainHeight,
		GenesisBlockAddress:      withdrawBank,
		SideChainTransactionHash: sideChainTransactionHash}
	program := &pg.Program{redeemScript, nil}

	// Create attributes
	txAttr := tx.NewTxAttribute(tx.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	attributes := make([]*tx.TxAttribute, 0)
	attributes = append(attributes, &txAttr)

	return &tx.Transaction{
		TxType:        tx.WithdrawAsset,
		Payload:       txPayload,
		Attributes:    attributes,
		UTXOInputs:    txInputs,
		BalanceInputs: []*tx.BalanceTxInput{},
		Outputs:       txOutputs,
		Programs:      []*pg.Program{program},
		LockTime:      uint32(0),
	}, nil
}

func (mc *MainChainImpl) ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error) {

	var result []*DepositInfo

	switch payloadObj := txn.Payload.(type) {
	case *payload.TransferCrossChainAsset:
		for k, v := range payloadObj.AddressesMap {
			info := &DepositInfo{
				MainChainProgramHash: txn.Outputs[v].ProgramHash,
				TargetAddress:        k,
				Amount:               txn.Outputs[v].Value,
			}
			result = append(result, info)
		}
	default:
		return nil, errors.New("Invalid payload")
	}

	return result, nil
}

func (mc *MainChainImpl) syncChainData() {
	var chainHeight uint32
	var currentHeight uint32
	var needSync bool

	for {
		chainHeight, currentHeight, needSync = mc.needSyncBlocks()
		if !needSync {
			break
		}

		for currentHeight < chainHeight {
			block, err := rpc.GetBlockByHeight(currentHeight, config.Parameters.MainNode.Rpc)
			if err != nil {
				break
			}
			mc.processBlock(block)

			// Update wallet height
			currentHeight = DbCache.CurrentHeight(block.BlockData.Height + 1)

			fmt.Print(">")
		}
	}

	fmt.Print("\n")
}

func (mc *MainChainImpl) needSyncBlocks() (uint32, uint32, bool) {

	chainHeight, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return 0, 0, false
	}

	currentHeight := DbCache.CurrentHeight(QueryHeightCode)

	if currentHeight >= chainHeight {
		return chainHeight, currentHeight, false
	}

	return chainHeight, currentHeight, true
}

func (mc *MainChainImpl) getAvailableUTXOs(utxos []*AddressUTXO) []*AddressUTXO {
	var availableUTXOs []*AddressUTXO
	var currentHeight = DbCache.CurrentHeight(QueryHeightCode)
	for _, utxo := range utxos {
		if utxo.Input.Sequence > 0 {
			if utxo.Input.Sequence >= currentHeight {
				continue
			}
			utxo.Input.Sequence = math.MaxUint32 - 1
		}
		availableUTXOs = append(availableUTXOs, utxo)
	}
	return availableUTXOs
}

func (mc *MainChainImpl) containGenesisBlockAddress(address string) (string, bool) {
	for _, node := range config.Parameters.SideNodeList {
		if node.GenesisBlockAddress == address {
			return node.DestroyAddress, true
		}
	}
	return "", false
}

func (mc *MainChainImpl) processBlock(block *BlockInfo) {
	// Add UTXO to wallet address from transaction outputs
	for _, txn := range block.Transactions {

		// Add UTXOs to wallet address from transaction outputs
		for index, output := range txn.Outputs {
			if destroyAddress, ok := mc.containGenesisBlockAddress(output.Address); ok {
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
				addressUTXO := &AddressUTXO{
					Input:               input,
					Amount:              amount,
					GenesisBlockAddress: output.Address,
					DestroyAddress:      destroyAddress,
				}
				DbCache.AddAddressUTXO(addressUTXO)
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
			DbCache.DeleteUTXO(txInput)
		}
	}
}

func InitMainChain(arbitrator Arbitrator) error {
	currentArbitrator, ok := arbitrator.(*ArbitratorImpl)
	if !ok {
		return errors.New("Unknown arbitrator type.")
	}

	mainChainServer := &MainChainImpl{&DistributedNodeServer{P2pCommand: WithdrawCommand}}
	P2PClientSingleton.AddListener(mainChainServer)
	currentArbitrator.SetMainChain(mainChainServer)

	mainChainClient := &MainChainClientImpl{&DistributedNodeClient{P2pCommand: WithdrawCommand}}
	P2PClientSingleton.AddListener(mainChainClient)
	currentArbitrator.SetMainChainClient(mainChainClient)

	return nil
}
