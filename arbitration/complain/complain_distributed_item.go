package complain

import (
	. "Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/core/transaction"
	"io"
)

type ComplainItemImpl struct {
	UserAddress      string
	GenesisBlockHash string
	TransactionHash  Uint256
	IsFromMainBlock  bool

	RawTransaction *transaction.Transaction
}

func (item *ComplainItemImpl) Serialize(w io.Writer) error {
	return nil
}

func (item *ComplainItemImpl) Deserialize(r io.Reader) error {
	return nil
}

func (item *ComplainItemImpl) Verify() error {
	return nil
}
