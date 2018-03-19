package sidechain

import (
	"Elastos.ELA.Arbiter/crypto"
	"Elastos.ELA.Arbiter/common"
	. "Elastos.ELA.Arbiter/arbitration/base"
)

type SideChain interface {
	AccountListener

	GetKey() string
	GetNode() SideChainNode
	CreateDepositTransaction(target *crypto.PublicKey, information *SpvInformation) *TransactionInfo

	IsTransactionValid(transactionHash common.Uint256) (bool, error)

	ParseUserMainPublicKey(transactionHash common.Uint256) *crypto.PublicKey
}

type SideChainManager interface {

	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain
}