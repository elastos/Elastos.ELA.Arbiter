package mainchain

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	tr "Elastos.ELA.Arbiter/common/typeTransformation"
	"Elastos.ELA.Arbiter/core/asset"
	pg "Elastos.ELA.Arbiter/core/program"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/core/transaction/payload"
	"Elastos.ELA.Arbiter/crypto"
	spvCore "SPVWallet/core"
	spvTx "SPVWallet/core/transaction"
	spvMsg "SPVWallet/p2p/msg"
	spvWallet "SPVWallet/wallet"
	"bytes"
	"errors"
	"fmt"
)

var SystemAssetId = getSystemAssetId()

type OpCode byte

type MainChain interface {
	CreateWithdrawTransaction(withdrawBank string, target common.Uint168, txn *tx.Transaction) (*tx.Transaction, error)
	ParseUserSideChainHash(txn *TransactionInfo) (map[common.Uint168]common.Uint168, error)
	OnTransactionConfirmed(merkleBlock spvMsg.MerkleBlock, trans []spvTx.Transaction)
}

type MainChainImpl struct {
}

func createRedeemScript() ([]byte, error) {

	//TODO get arbitrators keys [jzh]
	//var arbitratorGroupImpl arbitrator.ArbitratorGroupImpl
	//arbitrators := arbitratorGroupImpl.GetArbitrators()
	//arbitratosPK := arbitrators.GetPK()
	arbitratosPK := []*crypto.PublicKey{}
	redeemScript, err := tx.CreateMultiSignRedeemScript(51, arbitratosPK)
	if err != nil {
		return nil, err
	}
	return redeemScript, nil
}

func getSystemAssetId() common.Uint256 {
	systemToken := &tx.Transaction{
		TxType:         tx.RegisterAsset,
		PayloadVersion: 0,
		Payload: &payload.RegisterAsset{
			Asset: &asset.Asset{
				Name:      "ELA",
				Precision: 0x08,
				AssetType: 0x00,
			},
			Amount:     0 * 100000000,
			Controller: common.Uint168{},
		},
		Attributes: []*tx.TxAttribute{},
		UTXOInputs: []*tx.UTXOTxInput{},
		Outputs:    []*tx.TxOutput{},
		Programs:   []*pg.Program{},
	}
	return systemToken.Hash()
}

func (mc *MainChainImpl) CreateWithdrawTransaction(withdrawBank string, target common.Uint168, txn *tx.Transaction) (*tx.Transaction, error) {

	amount := txn.Outputs[0].Value //TODO get amount [jzh]

	fromAddress := withdrawBank

	// Check if from address is valid
	spender, err := spvCore.Uint168FromAddress(fromAddress)
	if err != nil {
		return nil, errors.New(fmt.Sprint("Invalid spender address: ", fromAddress, ", error: ", err))
	}

	// Create transaction outputs
	var totalOutputAmount = spvCore.Fixed64(0) // The total amount will be spend
	var txOutputs []*tx.TxOutput               // The outputs in transaction
	//totalOutputAmount += *fee             // Add transaction fee
	txOutput := &tx.TxOutput{
		AssetID:     SystemAssetId,
		ProgramHash: target,
		Value:       amount,
		OutputLock:  uint32(0),
	}

	txOutputs = append(txOutputs, txOutput)

	// Get spender's UTXOs
	database, err := spvWallet.GetDatabase()
	if err != nil {
		return nil, errors.New("[Wallet], Get db failed")
	}
	utxos, err := database.GetAddressUTXOs(spender)
	if err != nil {
		return nil, errors.New("[Wallet], Get spender's UTXOs failed")
	}
	availableUTXOs := utxos
	//availableUTXOs := db.removeLockedUTXOs(UTXOs) // Remove locked UTXOs
	//availableUTXOs = SortUTXOs(availableUTXOs)    // Sort available UTXOs by value ASC

	// Create transaction inputs
	var txInputs []*tx.UTXOTxInput // The inputs in transaction
	for _, utxo := range availableUTXOs {
		txInputs = append(txInputs, tr.TxUTXOFromSpvUTXO(utxo))
		if utxo.Value < totalOutputAmount {
			totalOutputAmount -= utxo.Value
		} else if utxo.Value == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if utxo.Value > totalOutputAmount {
			programHash, err := common.Uint168FromAddress(fromAddress)
			if err != nil {
				return nil, err
			}
			change := &tx.TxOutput{
				AssetID:     SystemAssetId,
				Value:       common.Fixed64(utxo.Value - totalOutputAmount),
				OutputLock:  uint32(0),
				ProgramHash: *programHash,
			}
			txOutputs = append(txOutputs, change)
			totalOutputAmount = 0
			break
		}
	}

	if totalOutputAmount > 0 {
		return nil, errors.New("Available token is not enough")
	}

	//get redeemscript
	redeemScript, err := createRedeemScript()
	if err != nil {
		return nil, err
	}

	// Create payload
	txPayload := &payload.TransferAsset{}
	// Create program
	program := &pg.Program{redeemScript, nil}

	return &tx.Transaction{
		TxType:        tx.TransferAsset,
		Payload:       txPayload,
		Attributes:    []*tx.TxAttribute{},
		UTXOInputs:    txInputs,
		BalanceInputs: []*tx.BalanceTxInput{},
		Outputs:       txOutputs,
		Programs:      []*pg.Program{program},
		LockTime:      uint32(0), //wallet.CurrentHeight(QueryHeightCode) - 1,
	}, nil
}

func (mc *MainChainImpl) ParseUserSideChainHash(txn *tx.Transaction) (map[common.Uint168]common.Uint168, error) {
	keyMap := make(map[common.Uint168]common.Uint168)
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
			attrIndex := txAttr.Data[len(txAttr.Data)-1 : len(txAttr.Data)]
			for index, output := range txn.Outputs {
				if bytes.Equal([]byte{byte(index)}, attrIndex) {
					keyMap[*targetProgramHash] = output.ProgramHash
					break
				}
			}
		}
	}

	return keyMap, nil
}

func (mc *MainChainImpl) OnTransactionConfirmed(merkleBlock spvMsg.MerkleBlock, trans []spvTx.Transaction) {

}
