package sideauxpow

import (
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"

	"github.com/elastos/Elastos.ELA/common"
	elacommon "github.com/elastos/Elastos.ELA/core/types/common"
)

const (
	DefaultKeystoreFile = "keystore.dat"
)

type Transfer struct {
	Address string
	Amount  *common.Fixed64
}

type UTXO struct {
	Op       *elacommon.OutPoint
	Amount   *common.Fixed64
	LockTime uint32
}

func GetAddressUTXOs(programHash *common.Uint168) ([]*UTXO, error) {
	address, err := programHash.ToAddress()
	if err != nil {
		return nil, err
	}

	utxoInfos, err := rpc.GetUnspentUtxo([]string{address}, config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}

	var inputs []*UTXO
	for _, utxoInfo := range utxoInfos {

		bytes, err := common.HexStringToBytes(utxoInfo.Txid)
		if err != nil {
			return nil, err
		}
		reversedBytes := common.BytesReverse(bytes)
		txid, err := common.Uint256FromBytes(reversedBytes)
		if err != nil {
			return nil, err
		}

		var op elacommon.OutPoint
		op.TxID = *txid
		op.Index = uint16(utxoInfo.VOut)

		amount, err := common.StringToFixed64(utxoInfo.Amount)
		if err != nil {
			return nil, err
		}

		inputs = append(inputs, &UTXO{&op, amount, utxoInfo.OutputLock})
	}
	return inputs, nil
}
