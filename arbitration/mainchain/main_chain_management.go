package mainchain

import (
	"Elastos.ELA.Arbiter/crypto"
	"Elastos.ELA.Arbiter/common"
	. "Elastos.ELA.Arbiter/arbitration/base"
)

type MainChain interface {
	AccountListener
	SpvValidation

	CreateWithdrawTransaction(withdrawBank *crypto.PublicKey, target *crypto.PublicKey) *TransactionInfo

	ParseSideChainKey(uint256 common.Uint256) *crypto.PublicKey
	ParseUserSidePublicKey(uint256 common.Uint256) *crypto.PublicKey
}