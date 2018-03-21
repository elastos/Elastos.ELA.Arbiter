package base

import "Elastos.ELA.Arbiter/common"

type SpvInformation struct {
	TransactionHash common.Uint256
	MerkleBranch    []common.Uint256
	Index           int
}

type SpvValidation interface {
	IsValid(information *SpvInformation) (bool, error)
	GenerateSpvInformation(transaction common.Uint256) *SpvInformation
}
