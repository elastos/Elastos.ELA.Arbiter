package complain

import (
	. "github.com/elastos/Elastos.ELA.Arbiter/common"
)

type ComplainItem struct {
	UserAddress      string
	GenesisBlockHash string
	TransactionHash  Uint256
	IsFromMainBlock  bool
}
