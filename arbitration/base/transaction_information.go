package base

import "Elastos.ELA.Arbiter/common"

type WithdrawInfo struct {
	TargetProgramHash common.Uint168
	Amount            common.Fixed64
}

type DepositInfo struct {
	MainChainProgramHash common.Uint168
	TargetProgramHash    common.Uint168
	Amount               common.Fixed64
}
