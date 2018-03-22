package mainchain

import (
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/core/asset"
	pg "Elastos.ELA.Arbiter/core/program"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/core/transaction/payload"
	"Elastos.ELA.Arbiter/crypto"
	"SPVWallet/core"
	spvTx "SPVWallet/core/transaction"
	"SPVWallet/p2p/msg"
	"SPVWallet/wallet"
	"bytes"
	"errors"
	"fmt"
)

var SystemAssetId = getSystemAssetId()

type OpCode byte

type MainChain interface {
	CreateWithdrawTransaction(withdrawBank string, target common.Uint168) (*TransactionInfo, error)
	ParseUserSideChainHash(txn *tx.Transaction) (map[common.Uint168]common.Uint168, error)
	OnTransactionConfirmed(merkleBlock msg.MerkleBlock, trans []spvTx.Transaction)
}

type MainChainImpl struct {
}

func createRedeemScript() (string, error) {

	//TODO get arbitrators keys [jzh]
	//var arbitratorGroupImpl arbitrator.ArbitratorGroupImpl
	//arbitrators := arbitratorGroupImpl.GetArbitrators()
	//arbitratosPK := arbitrators.GetPK()
	arbitratosPK := []*crypto.PublicKey{}
	redeemScriptByte, err := tx.CreateMultiSignRedeemScript(51, arbitratosPK)
	if err != nil {
		return "", err
	}
	redeemScriptStr := common.BytesToHexString(redeemScriptByte)
	return redeemScriptStr, nil
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

func (mc *MainChainImpl) CreateWithdrawTransaction(withdrawBank string, target common.Uint168) (*TransactionInfo, error) {

	tx3 := TransactionInfo{} //TODO get tx3 [jzh]
	amount := tx3.Outputs[0].Value

	fromAddress := withdrawBank
	toAddress, err := target.ToAddress()
	if err != nil {
		return nil, errors.New("program hash  to address failed")
	}

	// Check if from address is valid
	spender, err := core.Uint168FromAddress(fromAddress)
	if err != nil {
		return nil, errors.New(fmt.Sprint("Invalid spender address: ", fromAddress, ", error: ", err))
	}

	// Create transaction outputs
	var totalOutputAmount = core.Fixed64(0) // The total amount will be spend
	var txOutputs []TxoutputInfo            // The outputs in transaction
	//totalOutputAmount += *fee             // Add transaction fee

	//receiver, err := common.Uint168FromAddress(toAddress)
	//if err != nil {
	//	return nil, errors.New(fmt.Sprint("Invalid receiver address: ", toAddress, ", error: ", err))
	//}
	txOutput := TxoutputInfo{
		AssetID:    SystemAssetId.String(),
		Address:    toAddress,
		Value:      amount,
		OutputLock: uint32(0),
	}

	txOutputs = append(txOutputs, txOutput)

	// Get spender's UTXOs
	database, err := wallet.GetDatabase()
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
	var txInputs []UTXOTxInputInfo // The inputs in transaction
	for _, utxo := range availableUTXOs {

		var input UTXOTxInputInfo
		input.ReferTxID = "" //common.BytesToHexString(utxo.Op.TxID.ToArrayReverse())
		input.ReferTxOutputIndex = utxo.Op.Index
		input.Sequence = utxo.LockTime
		input.Address = "" //prevOutput.ProgramHash.ToAddress()
		input.Value = ""   //prevOutput.Value.String()

		txInputs = append(txInputs, input)
		if utxo.Value < totalOutputAmount {
			totalOutputAmount -= utxo.Value
		} else if utxo.Value == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if utxo.Value > totalOutputAmount {
			change := TxoutputInfo{
				AssetID:    SystemAssetId.String(),
				Value:      (utxo.Value - totalOutputAmount).String(),
				OutputLock: uint32(0),
				Address:    fromAddress,
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
	txPayload := TransferAssetInfo{}
	// Create program
	var program = ProgramInfo{redeemScript, ""}

	return &TransactionInfo{
		TxType:        tx.TransferAsset,
		Payload:       txPayload,
		Attributes:    []TxAttributeInfo{},
		UTXOInputs:    txInputs,
		BalanceInputs: []BalanceTxInputInfo{},
		Outputs:       txOutputs,
		Programs:      []ProgramInfo{program},
		LockTime:      uint32(0), //wallet.CurrentHeight(QueryHeightCode) - 1,
	}, nil
}

func (mc *MainChainImpl) ParseUserSideChainHash(txn *tx.Transaction) (map[common.Uint168]common.Uint168, error) {

	//TODO get Transaction by hash [jzh]
	//var txn tx.Transaction
	//1.get Transaction by hash

	//2.getPublicKey from Transaction
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

func (mc *MainChainImpl) OnTransactionConfirmed(merkleBlock msg.MerkleBlock, trans []spvTx.Transaction) {

}
