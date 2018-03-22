package sidechain

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	tr "Elastos.ELA.Arbiter/common/typeTransformation"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/crypto"
	spvMsg "SPVWallet/p2p/msg"
	spvWallet "SPVWallet/wallet"
)

type SideChain interface {
	AccountListener

	GetKey() string
	GetNode() SideChainNode
	CreateDepositTransaction(target common.Uint168, merkleBlock spvMsg.MerkleBlock, amount common.Fixed64) (*TransactionInfo, error)

	IsTransactionValid(transactionHash common.Uint256) (bool, error)

	ParseUserMainChainHash(txn *tx.Transaction) ([]common.Uint168, error)
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain
}

type SideChainImpl struct {
	AccountListener
}

func (sc *SideChainImpl) GetKey() string {
	return ""
}

func (sc *SideChainImpl) GetNode() SideChainNode {
	return nil
}

func (sc *SideChainImpl) OnUTXOChanged(txinfo *TransactionInfo) error {
	//TODO　verify tx [jzh]

	txn, err := tr.TransactionFromTransactionInfo(txinfo)
	if err != nil {
		return err
	}
	targetHashList, err := sc.ParseUserMainChainHash(txn)
	if err != nil {
		return err
	}
	/*	for _, hashA := range targetHashList {
		currentArbitrator, err := arbitratorgroup.ArbitratorGroupSingleton.GetCurrentArbitrator()
		if err != nil {
			return err
		}
		tx3, err := tr.TransactionFromTransactionInfo(txinfo)
		tx4, err := currentArbitrator.CreateWithdrawTransaction(sideChain.GetKey(), hashA, tx3)
		buf := new(bytes.Buffer)
		err = tx4.Serialize(buf)
		if err != nil {
			return err
		}
		currentArbitrator.GetArbitrationNet().Broadcast(buf.Bytes())
	}*/

	if len(targetHashList) == 0 {
		return nil
	}
	return nil
}

func (sc *SideChainImpl) CreateDepositTransaction(target common.Uint168, merkleBlock spvMsg.MerkleBlock, amount common.Fixed64) (*TransactionInfo, error) {
	var totalOutputAmount = amount // The total amount will be spend
	var txOutputs []TxoutputInfo   // The outputs in transaction

	toAddress, err := target.ToAddress()
	if err != nil {
		return nil, err
	}

	assetID := spvWallet.SystemAssetId
	txOutput := TxoutputInfo{
		AssetID:    assetID.String(),
		Value:      totalOutputAmount.String(),
		Address:    toAddress,
		OutputLock: uint32(0),
	}
	txOutputs = append(txOutputs, txOutput)

	// Create payload
	txPayloadInfo := TransferAssetInfo{}

	// Create attributes
	spvInfo, err := merkleBlock.Serialize()
	if err != nil {
		return nil, err
	}
	txAttr := TxAttributeInfo{tx.SpvInfo, common.BytesToHexString(spvInfo)}
	attributes := make([]TxAttributeInfo, 0)
	attributes = append(attributes, txAttr)

	// Create program
	program := ProgramInfo{}
	return &TransactionInfo{
		TxType:        tx.IssueToken,
		Payload:       txPayloadInfo,
		Attributes:    attributes,
		UTXOInputs:    []UTXOTxInputInfo{},
		BalanceInputs: []BalanceTxInputInfo{},
		Outputs:       txOutputs,
		Programs:      []ProgramInfo{program},
		LockTime:      uint32(0),
	}, nil
}

func (sc *SideChainImpl) IsTransactionValid(transactionHash common.Uint256) (bool, error) {
	return false, nil
}

func (sc *SideChainImpl) ParseUserMainChainHash(txn *tx.Transaction) ([]common.Uint168, error) {

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
