package mainchain

import (
	. "Elastos.ELA.Arbiter/core/transaction"
	"io"
)

type TransactionDistributedItem struct {
	RawTransaction *Transaction
}

func (item *TransactionDistributedItem) Serialize(w io.Writer) error {
	return item.RawTransaction.SerializeUnsigned(w)
}

func (item *TransactionDistributedItem) Deserialize(r io.Reader) error {
	return item.RawTransaction.DeserializeUnsigned(r)
}
