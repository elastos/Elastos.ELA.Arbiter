package sidechain

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/crypto"
)

type SideChain interface {
	AccountListener

	GetKey() string
	GetNode() SideChainNode
	CreateDepositTransaction(target common.Uint168, information *SpvInformation) (*TransactionInfo, error)

	IsTransactionValid(transactionHash common.Uint256) (bool, error)

	ParseUserMainChainKey(hash common.Uint256) ([]common.Uint168, error)
}

type SideChainImpl struct {
	AccountListener
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain
}

func (sideChain *SideChainImpl) CreateDepositTransaction(target common.Uint168, information *SpvInformation) (*TransactionInfo, error) {
	// Create transaction outputs
	// TODO heropan
	var totalOutputAmount = common.Fixed64(0) // The total amount will be spend
	var txOutputs []TxoutputInfo              // The outputs in transaction

	toAddress, err := target.ToAddress()
	if err != nil {
		return nil, err
	}

	txOutput := TxoutputInfo{
		AssetID:    "AssetID", // TODO heropan
		Value:      totalOutputAmount.String(),
		Address:    toAddress,
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
	}, nil
}

func (sc *SideChainImpl) GetKey() string {
	return ""
}

func (sc *SideChainImpl) ParseUserMainChainKey(hash common.Uint256) ([]common.Uint168, error) {

	//TODO get Transaction by hash [jzh]
	var txn tx.Transaction
	//1.get Transaction by hash

	//2.getPublicKey from Transaction
	hashes := []common.Uint168{}
	txAttribute := txn.Attributes
	for _, txAttr := range txAttribute {
		if txAttr.Usage == tx.TargetPublicKey {
			// Get public key
			keyBytes := txAttr.Data[0 : len(txAttr.Data)-1]
			pka, err := crypto.DecodePoint(keyBytes)
			if err != nil {
				return nil, err
			}
			targetRedeemScript, err := tx.CreateStandardRedeemScript(pka)
			if err != nil {
				return nil, err
			}
			targetProgramHash, err := tx.ToProgramHash(targetRedeemScript)
			if err != nil {
				return nil, err
			}
			hashes = append(hashes, *targetProgramHash)
		}
	}

	return hashes, nil
}
