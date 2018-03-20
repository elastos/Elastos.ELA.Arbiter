package sidechain

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/core/transaction/payload"
	"Elastos.ELA.Arbiter/crypto"
)

type SideChain interface {
	AccountListener

	GetKey() string
	GetNode() SideChainNode
	CreateDepositTransaction(target *crypto.PublicKey, information *SpvInformation) *TransactionInfo

	IsTransactionValid(transactionHash common.Uint256) (bool, error)

	ParseUserMainPublicKey(transactionHash common.Uint256) *crypto.PublicKey
}

type SideChainImpl struct {
	AccountListener
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain
}

func (sideChain *SideChainImpl) CreateDepositTransaction(target *crypto.PublicKey, information *SpvInformation) *TransactionInfo {
	// Create transaction outputs
	// TODO heropan
	var totalOutputAmount = common.Fixed64(0) // The total amount will be spend
	var txOutputs []TxoutputInfo              // The outputs in transaction

	publicKeyBytes, _ := target.EncodePoint(true)

	txOutput := TxoutputInfo{
		AssetID:    "AssetID", // TODO heropan
		Value:      totalOutputAmount.String(),
		Address:    common.BytesToHexString(publicKeyBytes),
		OutputLock: uint32(0),
	}
	txOutputs = append(txOutputs, txOutput)

	// Create payload
	txPayloadInfo := TransferAssetInfo{}
	// Create attributes
	txAttr := TxAttributeInfo{tx.SpvInfo, "spvinformation"} // TODO heropan spvinformation
	attributes := make([]TxAttributeInfo, 0)
	attributes = append(attributes, txAttr)
	// Create program
	var program = ProgramInfo{"redeemScript", nil} // TODO heropan add redeemScript later
	return &TransactionInfo{
		TxType:        tx.IssueToken,
		Payload:       txPayloadInfo,
		Attributes:    attributes,
		UTXOInputs:    []UTXOTxInputInfo{},
		BalanceInputs: []BalanceTxInputInfo{},
		Outputs:       txOutputs,
		Programs:      []ProgramInfo{program},
		LockTime:      uint32(0), //wallet.CurrentHeight(QueryHeightCode) - 1,
	}
}
