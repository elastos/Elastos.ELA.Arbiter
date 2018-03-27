package mainchain

import (
	. "Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "Elastos.ELA.Arbiter/arbitration/base"
	. "Elastos.ELA.Arbiter/arbitration/cs"
	"Elastos.ELA.Arbiter/common"
	pg "Elastos.ELA.Arbiter/core/program"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/core/transaction/payload"
	"Elastos.ELA.Arbiter/crypto"
	spvCore "SPVWallet/core"
	spvtx "SPVWallet/core/transaction"
	spvdb "SPVWallet/db"
	spvWallet "SPVWallet/wallet"
	"bytes"
	"errors"
	"fmt"
)

type MainChainImpl struct {
	*DistributedNodeServer
}

func (mc *MainChainImpl) CreateWithdrawTransaction(withdrawBank string, target common.Uint168, amount common.Fixed64) (*tx.Transaction, error) {
	// Check if from address is valid
	spender, err := spvCore.Uint168FromAddress(withdrawBank)
	if err != nil {
		return nil, errors.New(fmt.Sprint("Invalid spender address: ", withdrawBank, ", error: ", err))
	}

	assetID := spvWallet.SystemAssetId

	// Create transaction outputs
	var totalOutputAmount = spvCore.Fixed64(0)
	var txOutputs []*tx.TxOutput
	txOutput := &tx.TxOutput{
		AssetID:     common.Uint256(assetID),
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
	var txInputs []*tx.UTXOTxInput
	for _, utxo := range availableUTXOs {
		txInputs = append(txInputs, TxUTXOFromSpvUTXO(utxo))
		if utxo.Value < totalOutputAmount {
			totalOutputAmount -= utxo.Value
		} else if utxo.Value == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if utxo.Value > totalOutputAmount {
			programHash, err := common.Uint168FromAddress(withdrawBank)
			if err != nil {
				return nil, err
			}
			change := &tx.TxOutput{
				AssetID:     common.Uint256(assetID),
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

	redeemScript, err := CreateRedeemScript()
	if err != nil {
		return nil, err
	}
	txPayload := &payload.TransferAsset{}
	program := &pg.Program{redeemScript, nil}

	return &tx.Transaction{
		TxType:        tx.TransferAsset,
		Payload:       txPayload,
		Attributes:    []*tx.TxAttribute{},
		UTXOInputs:    txInputs,
		BalanceInputs: []*tx.BalanceTxInput{},
		Outputs:       txOutputs,
		Programs:      []*pg.Program{program},
		LockTime:      uint32(0),
	}, nil
}

func (mc *MainChainImpl) ParseUserDepositTransactionInfo(txn *tx.Transaction) ([]*DepositInfo, error) {

	var result []*DepositInfo
	txAttribute := txn.Attributes
	for _, txAttr := range txAttribute {
		if txAttr.Usage == tx.TargetPublicKey {
			// Get public key
			keyBytes := txAttr.Data[0 : len(txAttr.Data)-1]
			key, err := crypto.DecodePoint(keyBytes)
			if err != nil {
				return nil, err
			}
			targetProgramHash, err := StandardAcccountPublicKeyToProgramHash(key)
			if err != nil {
				return nil, err
			}
			attrIndex := txAttr.Data[len(txAttr.Data)-1 : len(txAttr.Data)]
			for index, output := range txn.Outputs {
				if bytes.Equal([]byte{byte(index)}, attrIndex) {
					info := &DepositInfo{
						MainChainProgramHash: output.ProgramHash,
						TargetProgramHash:    *targetProgramHash,
						Amount:               output.Value,
					}
					result = append(result, info)
					break
				}
			}
		}
	}

	return result, nil
}

func (mc *MainChainImpl) OnTransactionConfirmed(proof spvdb.Proof, spvtxn spvtx.Transaction) {

}

func InitMainChain(arbitrator Arbitrator) error {
	currentArbitrator, ok := arbitrator.(*ArbitratorImpl)
	if !ok {
		return errors.New("Unknown arbitrator type.")
	}

	mainChainServer := &MainChainImpl{&DistributedNodeServer{P2pCommand: WithdrawCommand}}
	P2PClientSingleton.AddListener(mainChainServer)
	currentArbitrator.SetMainChain(mainChainServer)

	mainChainClient := &MainChainClientImpl{&DistributedNodeClient{P2pCommand: WithdrawCommand}}
	P2PClientSingleton.AddListener(mainChainClient)
	currentArbitrator.SetMainChainClient(mainChainClient)

	return nil
}
