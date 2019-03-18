package sideauxpow

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
)

func createTransaction(txType types.TxType, txPayload types.Payload, fromAddress string, fee *common.Fixed64, redeemScript []byte, lockedUntil uint32, currentHeight uint32, outputs ...*Transfer) (*types.Transaction, error) {
	// Check if output is valid
	if len(outputs) == 0 {
		return nil, errors.New("[Wallet], Invalid transaction target")
	}
	// Check if from address is valid
	spender, err := common.Uint168FromAddress(fromAddress)
	if err != nil {
		return nil, errors.New(fmt.Sprint("[Wallet], Invalid spender address: ", fromAddress, ", error: ", err))
	}
	// Create transaction outputs
	var totalOutputAmount = common.Fixed64(0) // The total amount will be spend
	var txOutputs []*types.Output             // The outputs in transaction
	totalOutputAmount += *fee                 // Add transaction fee

	for _, output := range outputs {
		receiver, err := common.Uint168FromAddress(output.Address)
		if err != nil {
			return nil, errors.New(fmt.Sprint("[Wallet], Invalid receiver address: ", output.Address, ", error: ", err))
		}
		txOutput := &types.Output{
			AssetID:     base.SystemAssetId,
			ProgramHash: *receiver,
			Value:       *output.Amount,
			OutputLock:  lockedUntil,
		}
		totalOutputAmount += *output.Amount
		txOutputs = append(txOutputs, txOutput)
	}
	// Get spender's UTXOs
	UTXOs, err := GetAddressUTXOs(spender)
	if err != nil {
		return nil, errors.New("[Wallet], Get spender's UTXOs failed")
	}
	availableUTXOs := removeLockedUTXOs(UTXOs, currentHeight) // Remove locked UTXOs
	availableUTXOs = SortUTXOs(availableUTXOs)                       // Sort available UTXOs by value ASC

	// Create transaction inputs
	var txInputs []*types.Input // The inputs in transaction
	for _, utxo := range availableUTXOs {
		input := &types.Input{
			Previous: types.OutPoint{
				TxID:  utxo.Op.TxID,
				Index: utxo.Op.Index,
			},
			Sequence: utxo.LockTime,
		}
		txInputs = append(txInputs, input)
		if *utxo.Amount < totalOutputAmount {
			totalOutputAmount -= *utxo.Amount
		} else if *utxo.Amount == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if *utxo.Amount > totalOutputAmount {
			change := &types.Output{
				AssetID:     base.SystemAssetId,
				Value:       *utxo.Amount - totalOutputAmount,
				OutputLock:  uint32(0),
				ProgramHash: *spender,
			}
			txOutputs = append(txOutputs, change)
			totalOutputAmount = 0
			break
		}
	}
	if totalOutputAmount > 0 {
		return nil, errors.New("[Wallet], Available token is not enough")
	}

	return newTransaction(txType, txPayload, redeemScript, txInputs, txOutputs, currentHeight), nil
}

func newTransaction(txType types.TxType, txPayload types.Payload, redeemScript []byte, inputs []*types.Input, outputs []*types.Output, currentHeight uint32) *types.Transaction {
	// Create attributes
	txAttr := types.NewAttribute(types.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	attributes := make([]*types.Attribute, 0)
	attributes = append(attributes, &txAttr)
	// Create program
	var p = &program.Program{redeemScript, nil}
	// Create transaction
	return &types.Transaction{
		TxType:     txType,
		Payload:    txPayload,
		Attributes: attributes,
		Inputs:     inputs,
		Outputs:    outputs,
		Programs:   []*program.Program{p},
		LockTime:   currentHeight - 1,
	}
}

func removeLockedUTXOs(utxos []*UTXO, currentHeight uint32) []*UTXO {
	var availableUTXOs []*UTXO
	for _, utxo := range utxos {
		if utxo.LockTime > 0 {
			if utxo.LockTime >= currentHeight {
				continue
			}
			utxo.LockTime = math.MaxUint32 - 1
		}
		availableUTXOs = append(availableUTXOs, utxo)
	}
	return availableUTXOs
}

func createAuxpowTransaction(txType types.TxType, txPayload types.Payload, fromAddress string, fee *common.Fixed64, redeemScript []byte, currentHeight uint32) (*types.Transaction, error) {
	// Check if from address is valid
	spender, err := common.Uint168FromAddress(fromAddress)
	if err != nil {
		return nil, errors.New(fmt.Sprint("[Wallet], Invalid spender address: ", fromAddress, ", error: ", err))
	}
	// Create transaction outputs
	var totalOutputAmount = common.Fixed64(0) // The total amount will be spend
	var txOutputs []*types.Output             // The outputs in transaction
	totalOutputAmount += *fee                 // Add transaction fee

	// Get spender's UTXOs
	UTXOs, err := GetAddressUTXOs(spender)
	if err != nil {
		return nil, errors.New("[Wallet], Get spender's UTXOs failed")
	}
	availableUTXOs := removeLockedUTXOs(UTXOs, currentHeight) // Remove locked UTXOs
	availableUTXOs = SortUTXOs(availableUTXOs)                       // Sort available UTXOs by value ASC

	// Create transaction inputs
	var txInputs []*types.Input // The inputs in transaction
	for _, utxo := range availableUTXOs {
		input := &types.Input{
			Previous: types.OutPoint{
				TxID:  utxo.Op.TxID,
				Index: utxo.Op.Index,
			},
			Sequence: utxo.LockTime,
		}
		txInputs = append(txInputs, input)
		if *utxo.Amount < totalOutputAmount {
			totalOutputAmount -= *utxo.Amount
		} else if *utxo.Amount == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if *utxo.Amount > totalOutputAmount {
			change := &types.Output{
				AssetID:     base.SystemAssetId,
				Value:       *utxo.Amount - totalOutputAmount,
				OutputLock:  uint32(0),
				ProgramHash: *spender,
			}
			txOutputs = append(txOutputs, change)
			totalOutputAmount = 0
			break
		}
	}
	if totalOutputAmount > 0 {
		return nil, errors.New("[Wallet], Available token is not enough")
	}

	// Check if output is valid add output with 0 amount to from address
	if len(txOutputs) == 0 {
		txOutput := &types.Output{
			AssetID:     base.SystemAssetId,
			ProgramHash: *spender,
			Value:       common.Fixed64(0),
			OutputLock:  uint32(0),
		}
		txOutputs = append(txOutputs, txOutput)
	}

	return newTransaction(txType, txPayload, redeemScript, txInputs, txOutputs, currentHeight), nil
}
