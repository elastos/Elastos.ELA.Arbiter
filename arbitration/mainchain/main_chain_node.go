package mainchain

import (
	"Elastos.ELA.Arbiter/arbitration/base"
)

type MainChainNode interface {
	GetCurrentHeight() (uint32, error)
	GetBlockByHeight(height uint32) base.BlockInfo
}
