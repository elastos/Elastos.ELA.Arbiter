package base

import "github.com/elastos/Elastos.ELA.Utility/common"

type WithdrawInfo struct {
	TargetAddress string
	Amount        common.Fixed64
}

type DepositInfo struct {
	MainChainProgramHash common.Uint168
	TargetAddress        string
	Amount               common.Fixed64
}
