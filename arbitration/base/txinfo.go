package base

import (
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA.SPV/bloom"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
)

type WithdrawAsset struct {
	TargetAddress    string
	Amount           *common.Fixed64
	CrossChainAmount *common.Fixed64
}

type WithdrawInfo struct {
	WithdrawAssets []*WithdrawAsset
}

type WithdrawTx struct {
	Txid         *common.Uint256
	WithdrawInfo *WithdrawInfo
}

type SpvTransaction struct {
	MainChainTransaction *types.Transaction
	Proof                *bloom.MerkleProof
}

type MainChainTransaction struct {
	TransactionHash     string
	GenesisBlockAddress string
	Transaction         *types.Transaction
	Proof               *bloom.MerkleProof
}

type SideChainTransaction struct {
	TransactionHash     string
	GenesisBlockAddress string
	Transaction         []byte
	BlockHeight         uint32
}

func (info *WithdrawInfo) Serialize(w io.Writer) error {
	if err := common.WriteVarUint(w, uint64(len(info.WithdrawAssets))); err != nil {
		return errors.New("[Serialize] write len withdraw assets failed")
	}

	for _, withdraw := range info.WithdrawAssets {
		if err := common.WriteVarString(w, withdraw.TargetAddress); err != nil {
			return errors.New("[Serialize] write withdraw target address failed")
		}

		if err := common.WriteElements(w, withdraw.Amount, withdraw.CrossChainAmount); err != nil {
			return errors.New("[Serialize] write withdraw assets failed")
		}
	}

	return nil
}

func (info *WithdrawInfo) Deserialize(r io.Reader) error {
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return errors.New("[Deserialize] read len withdraw assets failed")
	}

	info.WithdrawAssets = make([]*WithdrawAsset, 0)
	for i := uint64(0); i < count; i++ {
		withdraw := &WithdrawAsset{
			Amount:           new(common.Fixed64),
			CrossChainAmount: new(common.Fixed64),
		}
		if withdraw.TargetAddress, err = common.ReadVarString(r); err != nil {
			return errors.New("[Deserialize] read withdraw target address failed")
		}

		if err := common.ReadElements(r, withdraw.Amount, withdraw.CrossChainAmount); err != nil {
			return errors.New("[Deserialize] read withdraw assets failed")
		}

		info.WithdrawAssets = append(info.WithdrawAssets, withdraw)
	}

	return nil
}

func (t *WithdrawTx) Serialize(w io.Writer) error {
	if err := common.WriteElement(w, t.Txid); err != nil {
		return errors.New("[Serialize] write txid failed")
	}

	if err := t.WithdrawInfo.Serialize(w); err != nil {
		return errors.New("[Serialize] write withdrawInfo failed")
	}

	return nil
}

func (t *WithdrawTx) Deserialize(r io.Reader) error {
	t.Txid = &common.Uint256{}
	if err := common.ReadElement(r, t.Txid); err != nil {
		return errors.New("[Deserialize] read txid failed")
	}

	t.WithdrawInfo = new(WithdrawInfo)
	if err := t.WithdrawInfo.Deserialize(r); err != nil {
		return errors.New("[Deserialize] read withdrawInfo failed")
	}

	return nil
}
