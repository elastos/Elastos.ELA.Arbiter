package mainchain

import (
	"Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/common/config"
	"Elastos.ELA.Arbiter/core/asset"
	pg "Elastos.ELA.Arbiter/core/program"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/core/transaction/payload"
	"Elastos.ELA.Arbiter/crypto"
	"Elastos.ELA.Arbiter/rpc"
	spvCore "SPVWallet/core"
	spvTx "SPVWallet/core/transaction"
	spvMsg "SPVWallet/p2p/msg"
	spvWallet "SPVWallet/wallet"
	"bytes"
	"errors"
	"fmt"
	"math"
)

const (
	TransactionAgreementRatio = 0.667 //over 2/3 of arbitrators agree to unlock the redeem script
)

var SystemAssetId = getSystemAssetId()

type OpCode byte

type MainChainImpl struct {
	unsolvedTransactions map[common.Uint256]*tx.Transaction
	finishedTransactions map[common.Uint256]bool
}

func createRedeemScript() ([]byte, error) {
	arbitratorCount := arbitrator.ArbitratorGroupSingleton.GetArbitratorsCount()
	publicKeys := make([]*crypto.PublicKey, arbitratorCount)
	for _, arStr := range arbitrator.ArbitratorGroupSingleton.GetAllArbitrators() {
		temp := &crypto.PublicKey{}
		temp.FromString(arStr)
		publicKeys = append(publicKeys, temp)
	}
	redeemScript, err := tx.CreateWithdrawRedeemScript(getTransactionAgreementArbitratorsCount(), publicKeys)
	if err != nil {
		return nil, err
	}
	return redeemScript, nil
}

func getTransactionAgreementArbitratorsCount() int {
	return int(math.Ceil(float64(arbitrator.ArbitratorGroupSingleton.GetArbitratorsCount()) * TransactionAgreementRatio))
}

func (mc *MainChainImpl) sendToArbitrator(otherArbitrator string, content []byte) error {
	//todo call p2p module to broadcast to other arbitrators
	return nil
}

func (mc *MainChainImpl) BroadcastWithdrawProposal(transaction *tx.Transaction) error {
	proposals, err := mc.generateWithdrawProposals(transaction)
	if err != nil {
		return err
	}

	for pkStr, content := range proposals {
		mc.sendToArbitrator(pkStr, content)
	}
	return nil
}

func (mc *MainChainImpl) generateWithdrawProposals(transaction *tx.Transaction) (map[string][]byte, error) {
	if _, ok := mc.unsolvedTransactions[transaction.Hash()]; ok {
		return nil, errors.New("Transaction already in process.")
	}
	mc.unsolvedTransactions[transaction.Hash()] = transaction

	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	if !currentArbitrator.IsOnDuty() {
		return nil, errors.New("Can not start a new proposal, you are not on duty.")
	}

	publicKeys := make(map[string]*crypto.PublicKey, arbitrator.ArbitratorGroupSingleton.GetArbitratorsCount())
	for _, arStr := range arbitrator.ArbitratorGroupSingleton.GetAllArbitrators() {
		temp := &crypto.PublicKey{}
		temp.FromString(arStr)
		publicKeys[arStr] = temp
	}

	results := make(map[string][]byte, len(publicKeys))
	for pkStr, pk := range publicKeys {
		programHash, err := StandardAcccountPublicKeyToProgramHash(pk)
		if err != nil {
			return nil, err
		}
		transactionItem := &DistributedTransactionItem{
			RawTransaction:              transaction,
			TargetArbitratorPublicKey:   pk,
			TargetArbitratorProgramHash: programHash,
		}
		transactionItem.InitScript(currentArbitrator)
		transactionItem.Sign(currentArbitrator)

		buf := new(bytes.Buffer)
		err = transactionItem.Serialize(buf)
		if err != nil {
			return nil, err
		}
		results[pkStr] = buf.Bytes()
	}
	return results, nil
}

func (mc *MainChainImpl) ReceiveProposalFeedback(content []byte) error {
	transactionItem := DistributedTransactionItem{}
	transactionItem.Deserialize(bytes.NewReader(content))
	newSign, err := transactionItem.ParseFeedbackSignedData()
	if err != nil {
		return err
	}

	txn, ok := mc.unsolvedTransactions[transactionItem.RawTransaction.Hash()]
	if !ok {
		errors.New("Can not find transaction.")
	}

	var signerIndex = -1
	programHashes, err := txn.GetMultiSignSigners()
	if err != nil {
		return err
	}
	userProgramHash := transactionItem.TargetArbitratorProgramHash
	for i, programHash := range programHashes {
		if *userProgramHash == *programHash {
			signerIndex = i
			break
		}
	}
	if signerIndex == -1 {
		return errors.New("Invalid multi sign signer")
	}

	signedCount, err := mc.mergeSignToTransaction(newSign, signerIndex, txn)
	if err != nil {
		return err
	}

	if signedCount >= getTransactionAgreementArbitratorsCount() {
		delete(mc.unsolvedTransactions, txn.Hash())

		content, err := mc.convertToTransactionContent(txn)
		if err != nil {
			mc.finishedTransactions[txn.Hash()] = false
			return err
		}

		result, err := rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("Data", content), config.Parameters.MainNode.Rpc)
		if err != nil {
			return err
		}
		mc.finishedTransactions[txn.Hash()] = true
		fmt.Println(result)
	}
	return nil
}

func (mc *MainChainImpl) convertToTransactionContent(txn *tx.Transaction) (string, error) {
	buf := new(bytes.Buffer)
	err := txn.Serialize(buf)
	if err != nil {
		return "", err
	}
	content := common.BytesToHexString(buf.Bytes())
	return content, nil
}

func (mc *MainChainImpl) mergeSignToTransaction(newSign []byte, signerIndex int, txn *tx.Transaction) (int, error) {
	param := txn.Programs[0].Parameter

	// Check if is first signature
	if param == nil {
		param = []byte{}
	} else {
		// Check if singer already signed
		publicKeys, err := txn.GetMultiSignPublicKeys()
		if err != nil {
			return 0, err
		}
		buf := new(bytes.Buffer)
		txn.SerializeUnsigned(buf)
		for i := 0; i < len(param); i += tx.SignatureScriptLength {
			// Remove length byte
			sign := param[i : i+tx.SignatureScriptLength][1:]
			publicKey := publicKeys[signerIndex][1:]
			pubKey, err := crypto.DecodePoint(publicKey)
			if err != nil {
				return 0, err
			}
			err = crypto.Verify(*pubKey, buf.Bytes(), sign)
			if err == nil {
				return 0, errors.New("signer already signed")
			}
		}
	}

	buf := new(bytes.Buffer)
	buf.Write(param)
	buf.Write(newSign)

	txn.Programs[0].Parameter = buf.Bytes()
	return len(txn.Programs[0].Parameter) / tx.SignatureScriptLength, nil
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

func (mc *MainChainImpl) CreateWithdrawTransaction(withdrawBank string, target common.Uint168, amount common.Fixed64) (*tx.Transaction, error) {
	// Check if from address is valid
	spender, err := spvCore.Uint168FromAddress(withdrawBank)
	if err != nil {
		return nil, errors.New(fmt.Sprint("Invalid spender address: ", withdrawBank, ", error: ", err))
	}

	// Create transaction outputs
	var totalOutputAmount = spvCore.Fixed64(0)
	var txOutputs []*tx.TxOutput
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

	redeemScript, err := createRedeemScript()
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

func (mc *MainChainImpl) OnTransactionConfirmed(merkleBlock spvMsg.MerkleBlock, trans []spvTx.Transaction) {

}
