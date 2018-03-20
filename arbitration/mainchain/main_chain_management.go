package mainchain

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/crypto"
)

type MainChain interface {
	SpvValidation

	CreateWithdrawTransaction(withdrawBank *crypto.PublicKey, target *crypto.PublicKey) *TransactionInfo

	ParseSideChainKey(uint256 common.Uint256) *crypto.PublicKey
	ParseUserSidePublicKey(uint256 common.Uint256) *crypto.PublicKey
}
