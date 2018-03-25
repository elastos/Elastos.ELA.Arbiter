package complain

import (
	. "Elastos.ELA.Arbiter/common"
	"io"
)

type ComplainItemImpl struct {
	UserAddress      string
	GenesisBlockHash string
	TransactionHash  Uint256
	IsFromMainBlock  bool

	redeemScript []byte
	signedData   []byte
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
