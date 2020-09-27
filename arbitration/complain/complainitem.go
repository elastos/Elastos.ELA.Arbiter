package complain

import (
	"github.com/elastos/Elastos.ELA/common"
)

type ComplainItem struct {
	UserAddress      string
	GenesisBlockHash string
	TransactionHash  common.Uint256
	IsFromMainBlock  bool
}
