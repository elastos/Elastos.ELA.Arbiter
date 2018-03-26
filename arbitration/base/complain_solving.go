package base

import (
	"Elastos.ELA.Arbiter/common"
)

type ComplainSolving interface {
	AcceptComplain(userAddress, genesisBlockHash string, transactionHash common.Uint256) ([]byte, error)
	BroadcastComplainSolving([]byte) error

	GetComplainStatus(transactionHash common.Uint256) uint
}
