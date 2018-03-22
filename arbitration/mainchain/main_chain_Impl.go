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
	"SPVWallet/core"
	"SPVWallet/wallet"
	"bytes"
	"errors"
	"fmt"
	"math"
)

var SystemAssetId = getSystemAssetId()

type MainChainImpl struct {
	AccountListener
	SpvValidation

	unsolvedTransactions map[common.Uint256]*tx.Transaction
	finishedTransactions map[common.Uint256]bool
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

func (mc *MainChainImpl) genereateProgramHash(key *crypto.PublicKey) *common.Uint168 {
	return nil
}

func (mc *MainChainImpl) sendToArbitrator(otherArbitrator string, content []byte) error {
	//todo call p2p module to broadcast to other arbitrators
	return nil
}

func (mc *MainChainImpl) BroadcastWithdrawProposal(password []byte) error {
	//todo create withdraw transaction
	var transaction *tx.Transaction

	if _, ok := mc.unsolvedTransactions[transaction.Hash()]; ok {
		return errors.New("Transaction already in process.")
	}
	mc.unsolvedTransactions[transaction.Hash()] = transaction

	currentArbitrator, err := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	if err != nil {
		return err
	}
	if !currentArbitrator.IsOnDuty() {
		return errors.New("Can not start a new proposal, you are not on duty.")
	}

	publicKeys := make(map[string]*crypto.PublicKey, arbitrator.ArbitratorGroupSingleton.GetArbitratorsCount())
	for _, arStr := range arbitrator.ArbitratorGroupSingleton.GetAllArbitrators() {
		temp := &crypto.PublicKey{}
		temp.FromString(arStr)
		publicKeys[arStr] = temp
	}

	for pkStr, pk := range publicKeys {
		transactionItem := &DistributedTransactionItem{
			RawTransaction:              transaction,
			TargetArbitratorPublicKey:   pk,
			TargetArbitratorProgramHash: mc.genereateProgramHash(pk),
		}
		transactionItem.InitScript(currentArbitrator)
		transactionItem.Sign(password, currentArbitrator)

		content, err := transactionItem.Serialize()
		if err != nil {
			return err
		}
		mc.sendToArbitrator(pkStr, content)
	}
	return nil
}

func (mc *MainChainImpl) ReceiveProposalFeedback(content []byte) error {
	transactionItem := DistributedTransactionItem{}
	transactionItem.Deserialize(content)
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

	if signedCount >= int(math.Ceil(float64(arbitrator.ArbitratorGroupSingleton.GetArbitratorsCount())*0.667)) {
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
	//todo convert transaction to rpc interface required string content
	return "", nil
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

func (mc *MainChainImpl) ParseUserSideChainHash(hash common.Uint256) (map[common.Uint168]common.Uint168, error) {

	//TODO get Transaction by hash [jzh]
	var txn tx.Transaction
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
