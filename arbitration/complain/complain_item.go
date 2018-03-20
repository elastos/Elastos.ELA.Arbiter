package complain

import (
	"Elastos.ELA.Arbiter/common"
)

type ComplainItem struct {
	UserAddress      string
	GenesisBlockHash string
	TransactionHash  common.Uint256
	IsFromMainBlock  bool
}

func (item *ComplainItem) Accepted() bool {
	return true
}

func (item *ComplainItem) Serialize() ([]byte, error) {
	return nil, nil
}

func (item *ComplainItem) Deserialize(content []byte) error {
	return nil
}
