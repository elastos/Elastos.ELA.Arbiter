package complain

import (
	. "Elastos.ELA.Arbiter/common"
)

type ComplainItem struct {
	UserAddress      string
	GenesisBlockHash string
	TransactionHash  Uint256
	IsFromMainBlock  bool
}
