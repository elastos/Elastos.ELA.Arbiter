package base

import (
	"errors"
	"io"

	. "github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA/core"
)

func (i *IssueTokenInfo) Data(version byte) string {
	return i.Proof
}

func (i *IssueTokenInfo) Serialize(w io.Writer, version byte) error {
	if err := WriteVarString(w, i.Proof); err != nil {
		return errors.New("Transaction IssueTokenInfo serialization failed.")
	}

	return nil
}

func (i *IssueTokenInfo) Deserialize(r io.Reader, version byte) error {
	value, err := ReadVarString(r)
	if err != nil {
		return errors.New("Transaction IssueTokenInfo deserialization failed.")
	}
	i.Proof = value

	return nil
}

func (i *RegisterAssetInfo) Data(version byte) string {
	return ""
}

func (i *RegisterAssetInfo) Serialize(w io.Writer, version byte) error {
	return nil
}

func (i *RegisterAssetInfo) Deserialize(r io.Reader, version byte) error {
	return nil
}

func (i *TransferAssetInfo) Data(version byte) string {
	return ""
}

func (i *TransferAssetInfo) Serialize(w io.Writer, version byte) error {
	return nil
}

func (i *TransferAssetInfo) Deserialize(r io.Reader, version byte) error {
	return nil
}

func (a *TransferCrossChainAssetInfo) Data(version byte) string {
	return ""
}

func (a *TransferCrossChainAssetInfo) Serialize(w io.Writer, version byte) error {
	if a.AddressesMap == nil {
		return errors.New("Invalid address map")
	}

	if err := WriteVarUint(w, uint64(len(a.AddressesMap))); err != nil {
		return errors.New("address map's length serialize failed")
	}

	for k, v := range a.AddressesMap {
		if err := WriteVarString(w, k); err != nil {
			return errors.New("address map's key serialize failed")
		}

		if err := WriteVarUint(w, v); err != nil {
			return errors.New("address map's value serialize failed")
		}
	}

	return nil
}

func (a *TransferCrossChainAssetInfo) Deserialize(r io.Reader, version byte) error {
	if a.AddressesMap == nil {
		return errors.New("Invalid address key map")
	}

	length, err := ReadVarUint(r, 0)
	if err != nil {
		return errors.New("address map's length deserialize failed")
	}

	a.AddressesMap = nil
	a.AddressesMap = make(map[string]uint64)
	for i := uint64(0); i < length; i++ {
		k, err := ReadVarString(r)
		if err != nil {
			return errors.New("address map's key deserialize failed")
		}

		v, err := ReadVarUint(r, 0)
		if err != nil {
			return errors.New("address map's value deserialize failed")
		}

		a.AddressesMap[k] = v
	}

	return nil
}

func (trans *TransactionInfo) ConvertFrom(tx *Transaction) error {
	return nil
}

func (trans *TransactionInfo) ConvertTo() (*Transaction, error) {
	return nil, nil
}

func PayloadInfoToTransPayload(p PayloadInfo) (Payload, error) {

	switch object := p.(type) {
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
	case *IssueTokenInfo:
		obj := new(PayloadIssueToken)
		proofBytes, err := HexStringToBytes(object.Proof)
		if err != nil {
			return nil, err
		}

		obj.MerkleProof = proofBytes
		return obj, nil
	case *TransferCrossChainAssetInfo:
		obj := new(PayloadTransferCrossChainAsset)
		obj.AddressesMap = object.AddressesMap
		return obj, nil
	}

	return nil, errors.New("Invalid payload type.")
}

func (txinfo *TransactionInfo) ToTransaction() (*Transaction, error) {

	txPaload, err := PayloadInfoToTransPayload(txinfo.Payload)
	if err != nil {
		return nil, err
	}

	var txAttribute []*Attribute
	for _, att := range txinfo.Attributes {
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
	for _, input := range txinfo.Inputs {
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
	for _, output := range txinfo.Outputs {
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
		programHash, err := Uint168FromAddress(output.Address)
		if err != nil {
			return nil, err
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
	for _, pgrm := range txinfo.Programs {
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
		TxType:         txinfo.TxType,
		PayloadVersion: txinfo.PayloadVersion,
		Payload:        txPaload,
		Attributes:     txAttribute,
		Inputs:         txUTXOTxInput,
		Outputs:        txOutputs,
		Programs:       txPrograms,
	}
	return txTransaction, nil
}
