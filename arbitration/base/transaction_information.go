package base

import (
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/bloom"
	"github.com/elastos/Elastos.ELA/core"
)

type WithdrawInfo struct {
	TargetAddress     []string
	Amount            []common.Fixed64
	CrossChainAmounts []common.Fixed64
}

type DepositInfo struct {
	MainChainProgramHash common.Uint168
	TargetAddress        []string
	Amount               []common.Fixed64
	CrossChainAmounts    []common.Fixed64
}

type SpvTransaction struct {
	MainChainTransaction *core.Transaction
	Proof                *bloom.MerkleProof
	DepositInfo          *DepositInfo
}
