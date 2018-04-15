package blockinfo

import (
	"io"

	. "github.com/elastos/Elastos.ELA.Arbiter/common"
	"github.com/elastos/Elastos.ELA.Arbiter/common/serialization"
)

type AuxPow struct {
	AuxMerkleBranch   []Uint256
	AuxMerkleIndex    int
	ParCoinbaseTx     BtcTx
	ParCoinBaseMerkle []Uint256
	ParMerkleIndex    int
	ParBlockHeader    BtcBlockHeader
	ParentHash        Uint256
}

func NewAuxPow(AuxMerkleBranch []Uint256, AuxMerkleIndex int,
	ParCoinbaseTx BtcTx, ParCoinBaseMerkle []Uint256,
	ParMerkleIndex int, ParBlockHeader BtcBlockHeader) *AuxPow {

	return &AuxPow{
		AuxMerkleBranch:   AuxMerkleBranch,
		AuxMerkleIndex:    AuxMerkleIndex,
		ParCoinbaseTx:     ParCoinbaseTx,
		ParCoinBaseMerkle: ParCoinBaseMerkle,
		ParMerkleIndex:    ParMerkleIndex,
		ParBlockHeader:    ParBlockHeader,
	}
}

func (ap *AuxPow) Serialize(w io.Writer) error {
	err := ap.ParCoinbaseTx.Serialize(w)
	if err != nil {
		return err
	}

	_, err = ap.ParentHash.Serialize(w)
	if err != nil {
		return err
	}

	count := uint64(len(ap.ParCoinBaseMerkle))
	err = serialization.WriteVarUint(w, count)
	if err != nil {
		return err
	}

	for _, pcbm := range ap.ParCoinBaseMerkle {
		_, err = pcbm.Serialize(w)
		if err != nil {
			return err
		}
	}
	idx := uint32(ap.ParMerkleIndex)
	err = serialization.WriteUint32(w, idx)
	if err != nil {
		return err
	}

	count = uint64(len(ap.AuxMerkleBranch))
	err = serialization.WriteVarUint(w, count)
	if err != nil {
		return err
	}

	for _, amb := range ap.AuxMerkleBranch {
		_, err = amb.Serialize(w)
		if err != nil {
			return err
		}
	}

	idx = uint32(ap.AuxMerkleIndex)
	err = serialization.WriteUint32(w, idx)
	if err != nil {
		return err
	}

	err = ap.ParBlockHeader.Serialize(w)
	if err != nil {
		return err
	}
	return nil
}

func (ap *AuxPow) Deserialize(r io.Reader) error {
	err := ap.ParCoinbaseTx.Deserialize(r)
	if err != nil {
		return err
	}

	err = ap.ParentHash.Deserialize(r)
	if err != nil {
		return err
	}

	count, err := serialization.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	ap.ParCoinBaseMerkle = make([]Uint256, count)
	for i := uint64(0); i < count; i++ {
		temp := Uint256{}
		err = temp.Deserialize(r)
		if err != nil {
			return err
		}
		ap.ParCoinBaseMerkle[i] = temp

	}

	temp, err := serialization.ReadUint32(r)
	if err != nil {
		return err
	}
	ap.ParMerkleIndex = int(temp)

	count, err = serialization.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	ap.AuxMerkleBranch = make([]Uint256, count)
	for i := uint64(0); i < count; i++ {
		temp := Uint256{}
		err = temp.Deserialize(r)
		if err != nil {
			return err
		}
		ap.AuxMerkleBranch[i] = temp
	}

	temp, err = serialization.ReadUint32(r)
	if err != nil {
		return err
	}
	ap.AuxMerkleIndex = int(temp)

	err = ap.ParBlockHeader.Deserialize(r)
	if err != nil {
		return err
	}

	return nil
}