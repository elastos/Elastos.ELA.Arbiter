package base

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/elastos/Elastos.ELA/auxpow"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

var SystemAssetId = getSystemAssetId()

func getSystemAssetId() common.Uint256 {
	systemToken := &types.Transaction{
		TxType:         types.RegisterAsset,
		PayloadVersion: 0,
		Payload: &payload.PayloadRegisterAsset{
			Asset: payload.Asset{
				Name:      "ELA",
				Precision: 0x08,
				AssetType: 0x00,
			},
			Amount:     0 * 100000000,
			Controller: common.Uint168{},
		},
		Attributes: []*types.Attribute{},
		Inputs:     []*types.Input{},
		Outputs:    []*types.Output{},
		Programs:   []*program.Program{},
	}
	return systemToken.Hash()
}

func PayloadInfoToTransPayload(plInfo PayloadInfo) (types.Payload, error) {

	switch object := plInfo.(type) {
	case *RegisterAssetInfo:
		obj := new(payload.PayloadRegisterAsset)
		obj.Asset = *object.Asset
		amount, err := common.StringToFixed64(object.Amount)
		if err != nil {
			return nil, err
		}
		obj.Amount = *amount
		bytes, err := common.HexStringToBytes(object.Controller)
		if err != nil {
			return nil, err
		}
		controller, err := common.Uint168FromBytes(bytes)
		obj.Controller = *controller
		return obj, nil
	case *TransferAssetInfo:
		return new(payload.PayloadTransferAsset), nil
		//fixme use transaction info instead of side chain payload
	//case *RechargeToSideChainInfoV0:
	//	obj := new(payload.PayloadRechargeToSideChain)
	//	proofBytes, err := common.HexStringToBytes(object.Proof)
	//	if err != nil {
	//		return nil, err
	//	}
	//	obj.MerkleProof = proofBytes
	//	transactionBytes, err := common.HexStringToBytes(object.MainChainTransaction)
	//	if err != nil {
	//		return nil, err
	//	}
	//	obj.MainChainTransaction = transactionBytes
	//	return obj, nil
	//case *RechargeToSideChainInfoV1:
	//	obj := new(payload.PayloadRechargeToSideChain)
	//	hash, err := common.Uint256FromHexString(object.MainChainTransactionHash)
	//	if err != nil {
	//		return nil, err
	//	}
	//	obj.MainChainTransactionHash = *hash
	//	return obj, nil
	case *TransferCrossChainAssetInfo:
		obj := new(payload.PayloadTransferCrossChainAsset)
		obj.CrossChainAddresses = make([]string, 0)
		obj.OutputIndexes = make([]uint64, 0)
		obj.CrossChainAmounts = make([]common.Fixed64, 0)
		for _, assetInfo := range object.CrossChainAssets {
			obj.CrossChainAddresses = append(obj.CrossChainAddresses, assetInfo.CrossChainAddress)
			obj.OutputIndexes = append(obj.OutputIndexes, assetInfo.OutputIndex)
			amount, err := common.StringToFixed64(assetInfo.CrossChainAmount)
			if err != nil {
				return nil, err
			}
			obj.CrossChainAmounts = append(obj.CrossChainAmounts, *amount)
		}
		return obj, nil
	}

	return nil, errors.New("Invalid payload type.")
}

func (txInfo *TransactionInfo) ToTransaction() (*types.Transaction, error) {

	txPaload, err := PayloadInfoToTransPayload(txInfo.Payload)
	if err != nil {
		return nil, err
	}

	var txAttribute []*types.Attribute
	for _, att := range txInfo.Attributes {
		var attData []byte
		if att.Usage == types.Nonce {
			attData = []byte(att.Data)
		} else {
			attData, err = common.HexStringToBytes(att.Data)
			if err != nil {
				return nil, err
			}
		}
		txAttr := &types.Attribute{
			Usage: att.Usage,
			Data:  attData,
		}
		txAttribute = append(txAttribute, txAttr)
	}

	var txUTXOTxInput []*types.Input
	for _, input := range txInfo.Inputs {
		txID, err := common.HexStringToBytes(input.TxID)
		if err != nil {
			return nil, err
		}
		referID, err := common.Uint256FromBytes(common.BytesReverse(txID))
		if err != nil {
			return nil, err
		}
		utxoInput := &types.Input{
			Previous: types.OutPoint{
				TxID:  *referID,
				Index: input.VOut,
			},
			Sequence: input.Sequence,
		}
		txUTXOTxInput = append(txUTXOTxInput, utxoInput)
	}

	var txOutputs []*types.Output
	for _, output := range txInfo.Outputs {
		value, err := common.StringToFixed64(output.Value)
		if err != nil {
			return nil, err
		}
		var programHash *common.Uint168
		if output.Address == DESTROY_ADDRESS {
			programHash = &common.Uint168{}
		} else {
			programHash, err = common.Uint168FromAddress(output.Address)
			if err != nil {
				return nil, err
			}
		}
		output := &types.Output{
			AssetID:     SystemAssetId,
			Value:       *value,
			OutputLock:  output.OutputLock,
			ProgramHash: *programHash,
		}
		txOutputs = append(txOutputs, output)
	}

	var txPrograms []*program.Program
	for _, pgrm := range txInfo.Programs {
		code, err := common.HexStringToBytes(pgrm.Code)
		if err != nil {
			return nil, err
		}
		parameter, err := common.HexStringToBytes(pgrm.Parameter)
		if err != nil {
			return nil, err
		}
		txProgram := &program.Program{
			Code:      code,
			Parameter: parameter,
		}
		txPrograms = append(txPrograms, txProgram)
	}

	txTransaction := &types.Transaction{
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

func GetBlockHeader(blInfo *BlockInfo) (*types.Header, error) {

	previousBytes, err := common.HexStringToBytes(blInfo.PreviousBlockHash)
	if err != nil {
		return nil, err
	}
	previous, err := common.Uint256FromBytes(previousBytes)
	if err != nil {
		return nil, err
	}

	merkleRootBytes, err := common.HexStringToBytes(blInfo.PreviousBlockHash)
	if err != nil {
		return nil, err
	}
	merkleRoot, err := common.Uint256FromBytes(merkleRootBytes)
	if err != nil {
		return nil, err
	}

	auxPowBytes, err := common.HexStringToBytes(blInfo.AuxPow)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(auxPowBytes)
	auxpow := new(auxpow.AuxPow)
	err = auxpow.Deserialize(reader)
	if err != nil {
		return nil, err
	}

	return &types.Header{
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

func (blInfo *BlockInfo) ToBlock() (*types.Block, error) {

	header, err := GetBlockHeader(blInfo)
	if err != nil {
		return nil, err
	}

	var transactions []*types.Transaction
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

	return &types.Block{
		Header:       *header,
		Transactions: transactions,
	}, nil
}
