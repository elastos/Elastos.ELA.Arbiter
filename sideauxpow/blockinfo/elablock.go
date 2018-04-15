package blockinfo

import (
	"errors"
	"io"

	. "github.com/elastos/Elastos.ELA.Arbiter/common"
	"github.com/elastos/Elastos.ELA.Arbiter/common/serialization"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
)

type Block struct {
	Blockdata    *Blockdata
	Transactions []*tx.Transaction

	hash *Uint256
}

func (b *Block) Serialize(w io.Writer) error {
	b.Blockdata.Serialize(w)
	err := serialization.WriteUint32(w, uint32(len(b.Transactions)))
	if err != nil {
		return errors.New("Block item Transactions length serialization failed.")
	}

	for _, transaction := range b.Transactions {
		transaction.Serialize(w)
	}
	return nil
}

func (b *Block) Deserialize(r io.Reader) error {
	if b.Blockdata == nil {
		b.Blockdata = new(Blockdata)
	}
	b.Blockdata.Deserialize(r)

	//Transactions
	var i uint32
	Len, err := serialization.ReadUint32(r)
	if err != nil {
		return err
	}
	var txhash Uint256
	var tharray []Uint256
	for i = 0; i < Len; i++ {
		transaction := new(tx.Transaction)
		transaction.Deserialize(r)
		txhash = transaction.Hash()
		b.Transactions = append(b.Transactions, transaction)
		tharray = append(tharray, txhash)
	}

	b.Blockdata.TransactionsRoot, err = sideauxpow.ComputeRoot(tharray)
	if err != nil {
		return errors.New("Block Deserialize merkleTree compute failed")
	}

	return nil
}
