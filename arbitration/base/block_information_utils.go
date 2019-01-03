package base

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/elastos/Elastos.ELA.SideChain/types"
	"github.com/elastos/Elastos.ELA/auxpow"
	. "github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	elat "github.com/elastos/Elastos.ELA/core/types"
	elap "github.com/elastos/Elastos.ELA/core/types/payload"
)

var SystemAssetId = getSystemAssetId()

func getSystemAssetId() Uint256 {
	systemToken := &elat.Transaction{
		TxType:         elat.RegisterAsset,
		PayloadVersion: 0,
		Payload: &elap.PayloadRegisterAsset{
			Asset: elap.Asset{
				Name:      "ELA",
				Precision: 0x08,
				AssetType: 0x00,
			},
			Amount:     0 * 100000000,
			Controller: Uint168{},
		},
		Attributes: []*elat.Attribute{},
		Inputs:     []*elat.Input{},
		Outputs:    []*elat.Output{},
		Programs:   []*program.Program{},
	}
	return systemToken.Hash()
}

func PayloadInfoToTransPayload(plInfo PayloadInfo) (elat.Payload, error) {

	switch object := plInfo.(type) {
	case *RegisterAssetInfo:
		obj := new(elap.PayloadRegisterAsset)
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
		return new(elap.PayloadTransferAsset), nil
	case *RechargeToSideChainInfoV0:
		obj := new(types.PayloadRechargeToSideChain)
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
	case *RechargeToSideChainInfoV1:
		obj := new(types.PayloadRechargeToSideChain)
		hash, err := Uint256FromHexString(object.MainChainTransactionHash)
		if err != nil {
			return nil, err
		}
		obj.MainChainTransactionHash = *hash
		return obj, nil
	case *TransferCrossChainAssetInfo:
		obj := new(elap.PayloadTransferCrossChainAsset)
		obj.CrossChainAddresses = make([]string, 0)
		obj.OutputIndexes = make([]uint64, 0)
		obj.CrossChainAmounts = make([]Fixed64, 0)
		for _, assetInfo := range object.CrossChainAssets {
			obj.CrossChainAddresses = append(obj.CrossChainAddresses, assetInfo.CrossChainAddress)
			obj.OutputIndexes = append(obj.OutputIndexes, assetInfo.OutputIndex)
			amount, err := StringToFixed64(assetInfo.CrossChainAmount)
			if err != nil {
				return nil, err
			}
			obj.CrossChainAmounts = append(obj.CrossChainAmounts, *amount)
		}
		return obj, nil
	}

	return nil, errors.New("Invalid payload type.")
}

func (txInfo *TransactionInfo) ToTransaction() (*elat.Transaction, error) {

	txPaload, err := PayloadInfoToTransPayload(txInfo.Payload)
	if err != nil {
		return nil, err
	}

	var txAttribute []*elat.Attribute
	for _, att := range txInfo.Attributes {
		var attData []byte
		if att.Usage == elat.Nonce {
			attData = []byte(att.Data)
		} else {
			attData, err = HexStringToBytes(att.Data)
			if err != nil {
				return nil, err
			}
		}
		txAttr := &elat.Attribute{
			Usage: att.Usage,
			Data:  attData,
		}
		txAttribute = append(txAttribute, txAttr)
	}

	var txUTXOTxInput []*elat.Input
	for _, input := range txInfo.Inputs {
		txID, err := HexStringToBytes(input.TxID)
		if err != nil {
			return nil, err
		}
		referID, err := Uint256FromBytes(BytesReverse(txID))
		if err != nil {
			return nil, err
		}
		utxoInput := &elat.Input{
			Previous: elat.OutPoint{
				TxID:  *referID,
				Index: input.VOut,
			},
			Sequence: input.Sequence,
		}
		txUTXOTxInput = append(txUTXOTxInput, utxoInput)
	}

	var txOutputs []*elat.Output
	for _, output := range txInfo.Outputs {
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
		output := &elat.Output{
			AssetID:     SystemAssetId,
			Value:       *value,
			OutputLock:  output.OutputLock,
			ProgramHash: *programHash,
		}
		txOutputs = append(txOutputs, output)
	}

	var txPrograms []*program.Program
	for _, pgrm := range txInfo.Programs {
		code, err := HexStringToBytes(pgrm.Code)
		if err != nil {
			return nil, err
		}
		parameter, err := HexStringToBytes(pgrm.Parameter)
		if err != nil {
			return nil, err
		}
		txProgram := &program.Program{
			Code:      code,
			Parameter: parameter,
		}
		txPrograms = append(txPrograms, txProgram)
	}

	txTransaction := &elat.Transaction{
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

func GetBlockHeader(blInfo *BlockInfo) (*elat.Header, error) {

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

	return &elat.Header{
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

func (blInfo *BlockInfo) ToBlock() (*elat.Block, error) {

	header, err := GetBlockHeader(blInfo)
	if err != nil {
		return nil, err
	}

	var transactions []*elat.Transaction
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

	return &elat.Block{
		Header:       *header,
		Transactions: transactions,
	}, nil
}
