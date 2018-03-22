package typeTransformation

import (
	"Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/core/program"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/core/transaction/payload"
	spvdb "SPVWallet/db"
)

func TransPayloadToHex(p tx.Payload) base.PayloadInfo {
	switch object := p.(type) {
	//case *payload.CoinBase:
	//	obj := new(CoinbaseInfo)
	//	obj.CoinbaseData = string(object.CoinbaseData)
	//	return obj
	case *payload.RegisterAsset:
		obj := new(base.RegisterAssetInfo)
		obj.Asset = object.Asset
		obj.Amount = object.Amount.String()
		obj.Controller = common.BytesToHexString(object.Controller.ToArrayReverse())
		return obj
	case *payload.TransferAsset:
	}
	return nil
}

func PayloadInfoToTransPayload(p base.PayloadInfo) (tx.Payload, error) {

	switch object := p.(type) {
	case *base.RegisterAssetInfo:
		obj := new(payload.RegisterAsset)
		obj.Asset = object.Asset
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
	case *base.TransferAssetInfo:
		return new(payload.TransferAsset), nil
	}
	return nil, nil
}

func TransactionFromTransactionInfo(txinfo *base.TransactionInfo) (*tx.Transaction, error) {

	txPaload, err := PayloadInfoToTransPayload(txinfo.Payload)
	if err != nil {
		return nil, err
	}

	var txAttribute []*tx.TxAttribute
	for _, att := range txinfo.Attributes {
		attData, err := common.HexStringToBytes(att.Data)
		if err != nil {
			return nil, err
		}
		txAttr := &tx.TxAttribute{
			Usage: att.Usage,
			Data:  attData,
			Size:  0,
		}
		txAttribute = append(txAttribute, txAttr)
	}

	var txUTXOTxInput []*tx.UTXOTxInput
	for _, input := range txinfo.UTXOInputs {
		txID, err := common.HexStringToBytes(input.ReferTxID)
		if err != nil {
			return nil, err
		}
		referID, err := common.Uint256FromBytes(txID)
		if err != nil {
			return nil, err
		}
		utxoInput := &tx.UTXOTxInput{
			ReferTxID:          *referID,
			ReferTxOutputIndex: input.ReferTxOutputIndex,
			Sequence:           input.Sequence,
		}
		txUTXOTxInput = append(txUTXOTxInput, utxoInput)
	}

	var txOutputs []*tx.TxOutput
	for _, output := range txinfo.Outputs {
		assetIdBytes, err := common.HexStringToBytes(output.AssetID)
		if err != nil {
			return nil, err
		}
		assetId, err := common.Uint256FromBytes(assetIdBytes)
		if err != nil {
			return nil, err
		}
		value, err := common.StringToFixed64(output.Value)
		if err != nil {
			return nil, err
		}
		programHash, err := common.Uint168FromAddress(output.Address)
		if err != nil {
			return nil, err
		}
		output := &tx.TxOutput{
			AssetID:     *assetId,
			Value:       *value,
			OutputLock:  output.OutputLock,
			ProgramHash: *programHash,
		}
		txOutputs = append(txOutputs, output)
	}

	var txPrograms []*program.Program
	for _, pgrm := range txinfo.Programs {
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

	txTransaction := &tx.Transaction{
		TxType:         txinfo.TxType,
		PayloadVersion: txinfo.PayloadVersion,
		Payload:        txPaload,
		Attributes:     txAttribute,
		UTXOInputs:     txUTXOTxInput,
		Outputs:        txOutputs,
		Programs:       txPrograms,
	}
	return txTransaction, nil
}

func TxUTXOFromSpvUTXO(utxo *spvdb.UTXO) *tx.UTXOTxInput {
	return &tx.UTXOTxInput{
		ReferTxID:          common.Uint256(utxo.Op.TxID),
		ReferTxOutputIndex: utxo.Op.Index,
		Sequence:           utxo.LockTime,
	}
}
