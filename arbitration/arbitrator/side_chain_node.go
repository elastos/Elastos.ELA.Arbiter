package arbitrator

import (
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Utility/common"
)

type SideChainNode interface {
	GetCurrentHeight() (uint32, error)
	GetBlockByHeight(height uint32) (*BlockInfo, error)

	SendTransaction(txHash *common.Uint256) (rpc.Response, error)
}
