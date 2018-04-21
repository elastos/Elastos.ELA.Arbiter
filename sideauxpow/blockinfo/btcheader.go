package blockinfo

import (
	"bytes"
	"crypto/sha256"
	"io"

	. "github.com/elastos/Elastos.ELA.Utility/common"
)

type BtcBlockHeader struct {
	Version    int32
	PrevBlock  Uint256
	MerkleRoot Uint256
	Timestamp  uint32
	Bits       uint32
	Nonce      uint32
}

func (bh *BtcBlockHeader) Serialize(w io.Writer) error {
	WriteUint32(w, uint32(bh.Version))
	bh.PrevBlock.Serialize(w)
	bh.MerkleRoot.Serialize(w)
	WriteUint32(w, bh.Timestamp)
	WriteUint32(w, bh.Bits)
	WriteUint32(w, bh.Nonce)
	return nil
}

func (bh *BtcBlockHeader) Deserialize(r io.Reader) error {
	//Version
	temp, err := ReadUint32(r)
	if err != nil {
		return err
	}
	bh.Version = int32(temp)

	//PrevBlockHash
	preBlock := new(Uint256)
	err = preBlock.Deserialize(r)
	if err != nil {
		return err
	}
	bh.PrevBlock = *preBlock

	//TransactionsRoot
	txRoot := new(Uint256)
	err = txRoot.Deserialize(r)
	if err != nil {
		return err
	}
	bh.MerkleRoot = *txRoot

	//Timestamp
	temp, _ = ReadUint32(r)
	bh.Timestamp = temp

	//Bits
	temp, _ = ReadUint32(r)
	bh.Bits = temp

	//Nonce
	bh.Nonce, _ = ReadUint32(r)

	return nil
}

func (bh *BtcBlockHeader) Hash() Uint256 {
	buf := new(bytes.Buffer)
	bh.Serialize(buf)
	temp := sha256.Sum256(buf.Bytes())
	return Uint256(sha256.Sum256(temp[:]))
}
