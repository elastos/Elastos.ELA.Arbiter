package base

import (
	"Elastos.ELA.Arbiter/common"
)

type ComplainSolving interface {
	AcceptComplain(userAddress, genesisBlockHash string, transactionHash common.Uint256) ([]byte, error)
	BroadcastComplainSolving([]byte) error

	GetComplainStatus(transactionHash common.Uint256) uint
}

type ComplainItem interface {
	GetUserAddress() string
	GetGenesisBlockHash() string
	GetTransactionHash() common.Uint256
	GetIsFromMainBlock() bool

	Accepted() bool
	Verify() bool
	Serialize() ([]byte, error)
	Deserialize(content []byte) error
}
