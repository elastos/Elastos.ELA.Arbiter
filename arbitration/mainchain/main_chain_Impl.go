package mainchain

import (
	. "Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "Elastos.ELA.Arbiter/arbitration/base"
	. "Elastos.ELA.Arbiter/arbitration/cs"
	. "Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/common/config"
	pg "Elastos.ELA.Arbiter/core/program"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/core/transaction/payload"
	"Elastos.ELA.Arbiter/crypto"
	"Elastos.ELA.Arbiter/rpc"
	. "Elastos.ELA.Arbiter/store"
	spvtx "SPVWallet/core/transaction"
	spvdb "SPVWallet/db"
	spvWallet "SPVWallet/wallet"
	"bytes"
	"errors"
	"fmt"
	"math"
)

type MainChainImpl struct {
	*DistributedNodeServer
}

func (mc *MainChainImpl) CreateWithdrawTransaction(withdrawBank string, target Uint168, amount Fixed64) (*tx.Transaction, error) {
	mc.syncChainData()

	// Check if from address is valid
	assetID := spvWallet.SystemAssetId

	// Create transaction outputs
	var totalOutputAmount = amount
	var txOutputs []*tx.TxOutput
	txOutput := &tx.TxOutput{
		AssetID:     Uint256(assetID),
		ProgramHash: target,
		Value:       amount,
		OutputLock:  uint32(0),
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
	txPayload := &payload.TransferAsset{}
	program := &pg.Program{redeemScript, nil}

	return &tx.Transaction{
		TxType:        tx.TransferAsset,
		Payload:       txPayload,
		Attributes:    []*tx.TxAttribute{},
		UTXOInputs:    txInputs,
		BalanceInputs: []*tx.BalanceTxInput{},
		Outputs:       txOutputs,
		Programs:      []*pg.Program{program},
		LockTime:      uint32(0),
	}, nil
}

func (mc *MainChainImpl) ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error) {

	var result []*DepositInfo
	txAttribute := txn.Attributes
	for _, txAttr := range txAttribute {
		if txAttr.Usage == tx.TargetPublicKey {
			// Get public key
			keyBytes := txAttr.Data[0 : len(txAttr.Data)-1]
			key, err := crypto.DecodePoint(keyBytes)
			if err != nil {
				return nil, err
			}
			targetProgramHash, err := StandardAcccountPublicKeyToProgramHash(key)
			if err != nil {
				return nil, err
			}
			attrIndex := txAttr.Data[len(txAttr.Data)-1 : len(txAttr.Data)]
			for index, output := range txn.Outputs {
				if bytes.Equal([]byte{byte(index)}, attrIndex) {
					info := &DepositInfo{
						MainChainProgramHash: output.ProgramHash,
						TargetProgramHash:    *targetProgramHash,
						Amount:               output.Value,
					}
					result = append(result, info)
					break
				}
			}
		}
	}

	return result, nil
}

func (mc *MainChainImpl) OnTransactionConfirmed(proof spvdb.Proof, spvtxn spvtx.Transaction) {
	//implement directly in arbitrator struct
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
			if genesisAddress, ok := mc.containGenesisBlockAddress(output.Address); ok {
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
					GenesisBlockAddress: genesisAddress,
					DestroyAddress:      output.Address,
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
