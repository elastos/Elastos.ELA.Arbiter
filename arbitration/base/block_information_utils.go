package base

import (
	"bytes"
	"encoding/json"
	"errors"

	sc "github.com/elastos/Elastos.ELA.SideChain/core"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/auxpow"
	. "github.com/elastos/Elastos.ELA/core"
)

func (txInfo *TransactionInfo) ConvertFrom(tx *Transaction) error {
	return nil
}

func (txInfo *TransactionInfo) ConvertTo() (*Transaction, error) {
	return nil, nil
}

func PayloadInfoToTransPayload(plInfo PayloadInfo) (Payload, error) {

	switch object := plInfo.(type) {
	case *RegisterAssetInfo:
		obj := new(PayloadRegisterAsset)
		obj.Asset = *object.Asset
		amount, err := StringToFixed64(object.Amount)
		if err != nil {
			return nil, err
		}
		obj.Amount = *amount
		bytes, err := HexStringToBytes(object.Controller)
		if err != nil {
			return nil, err
		}
		controller, err := Uint168FromBytes(bytes)
		obj.Controller = *controller
		return obj, nil
	case *TransferAssetInfo:
		return new(PayloadTransferAsset), nil
	case *RechargeToSideChainInfo:
		obj := new(sc.PayloadRechargeToSideChain)
		proofBytes, err := HexStringToBytes(object.Proof)
		if err != nil {
			return nil, err
		}
		obj.MerkleProof = proofBytes
		transactionBytes, err := HexStringToBytes(object.MainChainTransaction)
		if err != nil {
			return nil, err
		}
		obj.MainChainTransaction = transactionBytes
		return obj, nil
	case *TransferCrossChainAssetInfo:
		obj := new(PayloadTransferCrossChainAsset)
		obj.CrossChainAddresses = object.CrossChainAddresses
		obj.OutputIndexes = object.OutputIndexes
		obj.CrossChainAmounts = object.CrossChainAmounts
		return obj, nil
	}

	return nil, errors.New("Invalid payload type.")
}

func (txInfo *TransactionInfo) ToTransaction() (*Transaction, error) {

	txPaload, err := PayloadInfoToTransPayload(txInfo.Payload)
	if err != nil {
		return nil, err
	}

	var txAttribute []*Attribute
	for _, att := range txInfo.Attributes {
		attData, err := HexStringToBytes(att.Data)
		if err != nil {
			return nil, err
		}
		txAttr := &Attribute{
			Usage: att.Usage,
			Data:  attData,
			Size:  0,
		}
		txAttribute = append(txAttribute, txAttr)
	}

	var txUTXOTxInput []*Input
	for _, input := range txInfo.Inputs {
		txID, err := HexStringToBytes(input.TxID)
		if err != nil {
			return nil, err
		}
		referID, err := Uint256FromBytes(txID)
		if err != nil {
			return nil, err
		}
		utxoInput := &Input{
			Previous: OutPoint{
				TxID:  *referID,
				Index: input.VOut,
			},
			Sequence: input.Sequence,
		}
		txUTXOTxInput = append(txUTXOTxInput, utxoInput)
	}

	var txOutputs []*Output
	for _, output := range txInfo.Outputs {
		assetIdBytes, err := HexStringToBytes(output.AssetID)
		if err != nil {
			return nil, err
		}
		assetId, err := Uint256FromBytes(assetIdBytes)
		if err != nil {
			return nil, err
		}
		value, err := StringToFixed64(output.Value)
		if err != nil {
			return nil, err
		}
		var programHash *Uint168
		if output.Address == DESTROY_ADDRESS {
			programHash = &Uint168{}
		} else {
			programHash, err = Uint168FromAddress(output.Address)
			if err != nil {
				return nil, err
			}
		}
		output := &Output{
			AssetID:     *assetId,
			Value:       *value,
			OutputLock:  output.OutputLock,
			ProgramHash: *programHash,
		}
		txOutputs = append(txOutputs, output)
	}

	var txPrograms []*Program
	for _, pgrm := range txInfo.Programs {
		code, err := HexStringToBytes(pgrm.Code)
		if err != nil {
			return nil, err
		}
		parameter, err := HexStringToBytes(pgrm.Parameter)
		if err != nil {
			return nil, err
		}
		txProgram := &Program{
			Code:      code,
			Parameter: parameter,
		}
		txPrograms = append(txPrograms, txProgram)
	}

	txTransaction := &Transaction{
		TxType:         txInfo.TxType,
		PayloadVersion: txInfo.PayloadVersion,
		Payload:        txPaload,
		Attributes:     txAttribute,
		Inputs:         txUTXOTxInput,
		Outputs:        txOutputs,
		Programs:       txPrograms,
	}
	return txTransaction, nil
}

func GetBlockHeader(blInfo *BlockInfo) (*Header, error) {

	previousBytes, err := HexStringToBytes(blInfo.PreviousBlockHash)
	if err != nil {
		return nil, err
	}
	previous, err := Uint256FromBytes(previousBytes)
	if err != nil {
		return nil, err
	}

	merkleRootBytes, err := HexStringToBytes(blInfo.PreviousBlockHash)
	if err != nil {
		return nil, err
	}
	merkleRoot, err := Uint256FromBytes(merkleRootBytes)
	if err != nil {
		return nil, err
	}

	auxPowBytes, err := HexStringToBytes(blInfo.AuxPow)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(auxPowBytes)
	auxpow := new(auxpow.AuxPow)
	err = auxpow.Deserialize(reader)
	if err != nil {
		return nil, err
	}

	return &Header{
		Version:    blInfo.Version,
		Previous:   *previous,
		MerkleRoot: *merkleRoot,
		Timestamp:  blInfo.Time,
		Bits:       blInfo.Bits,
		Nonce:      blInfo.Nonce,
		Height:     blInfo.Height,
		AuxPow:     *auxpow,
	}, nil
}

func (blInfo *BlockInfo) ToBlock() (*Block, error) {

	header, err := GetBlockHeader(blInfo)
	if err != nil {
		return nil, err
	}

	var transactions []*Transaction
	for _, txInfo := range blInfo.Tx {
		switch txInfo.(type) {
		case *TransactionInfo:
			var tx TransactionInfo
			data, err := json.Marshal(&txInfo)
			if err != nil {
				return nil, err
			}
			err = json.Unmarshal(data, &tx)
			if err != nil {
				return nil, err
			}
			transaction, err := tx.ToTransaction()
			if err != nil {
				return nil, err
			}
			transactions = append(transactions, transaction)
		}
	}

	return &Block{
		Header:       *header,
		Transactions: transactions,
	}, nil
}
