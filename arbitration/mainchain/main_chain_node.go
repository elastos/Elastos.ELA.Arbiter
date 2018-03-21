package mainchain

import "Elastos.ELA.Arbiter/rpc"

type MainChainNode interface {
	GetCurrentHeight() (uint32, error)
	GetBlockByHeight(height uint32) rpc.BlockInfo
}
