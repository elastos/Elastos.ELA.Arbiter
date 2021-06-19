package arbitrator

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"

	"github.com/elastos/Elastos.ELA/common"
)

type SideChainNode interface {
	GetCurrentHeight() (uint32, error)
	GetBlockByHeight(height uint32) (*base.BlockInfo, error)
	GetCurrentConfig() *config.SideNodeConfig
	SendTransaction(txHash *common.Uint256) (rpc.Response, error)
	SendSmallCrossTransaction(tx string, signature []byte, hash string) (rpc.Response, error)
	IsSendSmallCrxTx(tx string) bool
	
	SendInvalidWithdrawTransaction(signature []byte, hash string) (rpc.Response, error)
}
