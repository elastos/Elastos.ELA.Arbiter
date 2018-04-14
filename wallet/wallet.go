package wallet

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"

	"github.com/elastos/Elastos.ELA.Arbiter/core/transaction"

	. "github.com/elastos/Elastos.ELA.Arbiter/common"
	"github.com/elastos/Elastos.ELA.Arbiter/common/log"
	"github.com/elastos/Elastos.ELA.Arbiter/core/asset"
	pg "github.com/elastos/Elastos.ELA.Arbiter/core/program"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
	"github.com/elastos/Elastos.ELA.Arbiter/core/transaction/payload"
	"github.com/elastos/Elastos.ELA.Arbiter/crypto"
)

var SystemAssetId = getSystemAssetId()

type Output struct {
	Address string
	Amount  *Fixed64
}

var wallet Wallet // Single instance of wallet

type Wallet interface {
	DataStore

	OpenKeystore(name string, password []byte) error
	ChangePassword(oldPassword, newPassword []byte) error

	AddStandardAccount(publicKey *crypto.PublicKey) (*Uint168, error)
	AddMultiSignAccount(M int, publicKey ...*crypto.PublicKey) (*Uint168, error)

	CreateTransaction(txType transaction.TransactionType, txPayload transaction.Payload, fromAddress, toAddress string, amount, fee *Fixed64) (*tx.Transaction, error)
	CreateLockedTransaction(txType transaction.TransactionType, txPayload transaction.Payload, fromAddress, toAddress string, amount, fee *Fixed64, lockedUntil uint32) (*tx.Transaction, error)
	CreateMultiOutputTransaction(fromAddress string, fee *Fixed64, output ...*Output) (*tx.Transaction, error)
	CreateLockedMultiOutputTransaction(txType transaction.TransactionType, txPayload transaction.Payload, fromAddress string, fee *Fixed64, lockedUntil uint32, output ...*Output) (*tx.Transaction, error)

	Sign(name string, password []byte, transaction *tx.Transaction) (*tx.Transaction, error)

	Reset() error
}

type WalletImpl struct {
	DataStore
	Keystore
}

func Create(name string, password []byte) (Wallet, error) {
	keyStore, err := CreateKeystore(name, password)
	if err != nil {
		log.Error("Wallet create key store failed:", err)
		return nil, err
	}

	dataStore, err := OpenDataStore()
	if err != nil {
		log.Error("Wallet create data store failed:", err)
		return nil, err
	}

	dataStore.AddAddress(keyStore.GetProgramHash(), keyStore.GetRedeemScript())

	wallet = &WalletImpl{
		DataStore: dataStore,
		Keystore:  keyStore,
	}
	return wallet, nil
}

func Open() (Wallet, error) {
	if wallet == nil {
		dataStore, err := OpenDataStore()
		if err != nil {
			return nil, err
		}

		wallet = &WalletImpl{
			DataStore: dataStore,
		}
	}
	return wallet, nil
}

func (wallet *WalletImpl) OpenKeystore(name string, password []byte) error {
	keyStore, err := OpenKeystore(name, password)
	if err != nil {
		return err
	}
	wallet.Keystore = keyStore
	return nil
}

func (wallet *WalletImpl) AddStandardAccount(publicKey *crypto.PublicKey) (*Uint168, error) {
	redeemScript, err := tx.CreateStandardRedeemScript(publicKey)
	if err != nil {
		return nil, errors.New("[Wallet], CreateStandardRedeemScript failed")
	}

	programHash, err := tx.ToProgramHash(redeemScript)
	if err != nil {
		return nil, errors.New("[Wallet], CreateStandardAddress failed")
	}

	err = wallet.AddAddress(programHash, redeemScript)
	if err != nil {
		return nil, err
	}

	return programHash, nil
}

func (wallet *WalletImpl) AddMultiSignAccount(M int, publicKeys ...*crypto.PublicKey) (*Uint168, error) {
	redeemScript, err := tx.CreateMultiSignRedeemScript(M, publicKeys)
	if err != nil {
		return nil, errors.New("[Wallet], CreateStandardRedeemScript failed")
	}

	programHash, err := tx.ToProgramHash(redeemScript)
	if err != nil {
		return nil, errors.New("[Wallet], CreateMultiSignAddress failed")
	}

	err = wallet.AddAddress(programHash, redeemScript)
	if err != nil {
		return nil, err
	}

	return programHash, nil
}

func (wallet *WalletImpl) CreateTransaction(txType transaction.TransactionType, txPayload transaction.Payload, fromAddress, toAddress string, amount, fee *Fixed64) (*tx.Transaction, error) {
	return wallet.CreateLockedTransaction(txType, txPayload, fromAddress, toAddress, amount, fee, uint32(0))
}

func (wallet *WalletImpl) CreateLockedTransaction(txType transaction.TransactionType, txPayload transaction.Payload, fromAddress, toAddress string, amount, fee *Fixed64, lockedUntil uint32) (*tx.Transaction, error) {
	return wallet.CreateLockedMultiOutputTransaction(txType, txPayload, fromAddress, fee, lockedUntil, &Output{toAddress, amount})
}

func (wallet *WalletImpl) CreateMultiOutputTransaction(fromAddress string, fee *Fixed64, outputs ...*Output) (*tx.Transaction, error) {
	txType := tx.TransferAsset
	txPayload := &payload.TransferAsset{}
	return wallet.CreateLockedMultiOutputTransaction(txType, txPayload, fromAddress, fee, uint32(0), outputs...)
}

func (wallet *WalletImpl) CreateLockedMultiOutputTransaction(txType transaction.TransactionType, txPayload transaction.Payload, fromAddress string, fee *Fixed64, lockedUntil uint32, outputs ...*Output) (*tx.Transaction, error) {
	return wallet.createTransaction(txType, txPayload, fromAddress, fee, lockedUntil, outputs...)
}

func (wallet *WalletImpl) createTransaction(txType transaction.TransactionType, txPayload transaction.Payload, fromAddress string, fee *Fixed64, lockedUntil uint32, outputs ...*Output) (*tx.Transaction, error) {
	// Check if output is valid
	if outputs == nil || len(outputs) == 0 {
		return nil, errors.New("[Wallet], Invalid transaction target")
	}
	// Sync chain block data before create transaction
	wallet.SyncChainData()

	// Check if from address is valid
	spender, err := Uint168FromAddress(fromAddress)
	if err != nil {
		return nil, errors.New(fmt.Sprint("[Wallet], Invalid spender address: ", fromAddress, ", error: ", err))
	}
	// Create transaction outputs
	var totalOutputAmount = Fixed64(0) // The total amount will be spend
	var txOutputs []*tx.TxOutput       // The outputs in transaction
	totalOutputAmount += *fee          // Add transaction fee

	for _, output := range outputs {
		receiver, err := Uint168FromAddress(output.Address)
		if err != nil {
			return nil, errors.New(fmt.Sprint("[Wallet], Invalid receiver address: ", output.Address, ", error: ", err))
		}
		txOutput := &tx.TxOutput{
			AssetID:     SystemAssetId,
			ProgramHash: *receiver,
			Value:       *output.Amount,
			OutputLock:  lockedUntil,
		}
		totalOutputAmount += *output.Amount
		txOutputs = append(txOutputs, txOutput)
	}
	// Get spender's UTXOs
	UTXOs, err := wallet.GetAddressUTXOs(spender)
	if err != nil {
		return nil, errors.New("[Wallet], Get spender's UTXOs failed")
	}
	availableUTXOs := wallet.removeLockedUTXOs(UTXOs) // Remove locked UTXOs
	availableUTXOs = SortUTXOs(availableUTXOs)        // Sort available UTXOs by value ASC

	// Create transaction inputs
	var txInputs []*tx.UTXOTxInput // The inputs in transaction
	for _, utxo := range availableUTXOs {
		input := &tx.UTXOTxInput{
			ReferTxID:          utxo.Op.TxID,
			ReferTxOutputIndex: utxo.Op.Index,
			Sequence:           utxo.LockTime,
		}
		txInputs = append(txInputs, input)
		if *utxo.Amount < totalOutputAmount {
			totalOutputAmount -= *utxo.Amount
		} else if *utxo.Amount == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if *utxo.Amount > totalOutputAmount {
			change := &tx.TxOutput{
				AssetID:     SystemAssetId,
				Value:       *utxo.Amount - totalOutputAmount,
				OutputLock:  uint32(0),
				ProgramHash: *spender,
			}
			txOutputs = append(txOutputs, change)
			totalOutputAmount = 0
			break
		}
	}
	if totalOutputAmount > 0 {
		return nil, errors.New("[Wallet], Available token is not enough")
	}

	account, err := wallet.GetAddressInfo(spender)
	if err != nil {
		return nil, errors.New("[Wallet], Get spenders account info failed")
	}

	return wallet.newTransaction(txType, txPayload, account.RedeemScript, txInputs, txOutputs), nil
}

func (wallet *WalletImpl) Sign(name string, password []byte, txn *tx.Transaction) (*tx.Transaction, error) {
	// Verify password
	err := wallet.OpenKeystore(name, password)
	if err != nil {
		return nil, err
	}
	// Get sign type
	signType, err := txn.GetTransactionType()
	if err != nil {
		return nil, err
	}
	// Look up transaction type
	if signType == tx.STANDARD {

		// Sign single transaction
		txn, err = wallet.signStandardTransaction(password, txn)
		if err != nil {
			return nil, err
		}

	} else if signType == tx.MULTISIG {

		// Sign multi sign transaction
		txn, err = wallet.signMultiSignTransaction(password, txn)
		if err != nil {
			return nil, err
		}
	}

	return txn, nil
}

func (wallet *WalletImpl) signStandardTransaction(password []byte, txn *tx.Transaction) (*tx.Transaction, error) {
	// Get signer
	programHash, err := txn.GetStandardSigner()
	// Check if current user is a valid signer
	if *programHash != *wallet.Keystore.GetProgramHash() {
		return nil, errors.New("[Wallet], Invalid signer")
	}
	// Sign transaction
	signedTx, err := wallet.Keystore.Sign(password, txn)
	if err != nil {
		return nil, err
	}
	// Add verify program for transaction
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(signedTx)))
	buf.Write(signedTx)
	// Add signature
	code, _ := txn.GetTransactionCode()
	var program = &pg.Program{code, buf.Bytes()}
	txn.SetPrograms([]*pg.Program{program})

	return txn, nil
}

func (wallet *WalletImpl) signMultiSignTransaction(password []byte, txn *tx.Transaction) (*tx.Transaction, error) {
	// Check if current user is a valid signer
	var signerIndex = -1
	programHashes, err := txn.GetMultiSignSigners()
	if err != nil {
		return nil, err
	}
	userProgramHash := wallet.Keystore.GetProgramHash()
	for i, programHash := range programHashes {
		if *userProgramHash == *programHash {
			signerIndex = i
			break
		}
	}
	if signerIndex == -1 {
		return nil, errors.New("[Wallet], Invalid multi sign signer")
	}
	// Sign transaction
	signedTx, err := wallet.Keystore.Sign(password, txn)
	if err != nil {
		return nil, err
	}
	// Append signature
	err = txn.AppendSignature(signerIndex, signedTx)
	if err != nil {
		return nil, err
	}

	return txn, nil
}

func (wallet *WalletImpl) Reset() error {
	return wallet.ResetDataStore()
}

func getSystemAssetId() Uint256 {
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
			Controller: Uint168{},
		},
		Attributes: []*tx.TxAttribute{},
		UTXOInputs: []*tx.UTXOTxInput{},
		Outputs:    []*tx.TxOutput{},
		Programs:   []*pg.Program{},
	}
	return systemToken.Hash()
}

func (wallet *WalletImpl) removeLockedUTXOs(utxos []*AddressUTXO) []*AddressUTXO {
	var availableUTXOs []*AddressUTXO
	var currentHeight = wallet.CurrentHeight(QueryHeightCode)
	for _, utxo := range utxos {
		if utxo.LockTime > 0 {
			if utxo.LockTime >= currentHeight {
				continue
			}
			utxo.LockTime = math.MaxUint32 - 1
		}
		availableUTXOs = append(availableUTXOs, utxo)
	}
	return availableUTXOs
}

func (wallet *WalletImpl) newTransaction(txType transaction.TransactionType, txPayload transaction.Payload, redeemScript []byte, inputs []*tx.UTXOTxInput, outputs []*tx.TxOutput) *tx.Transaction {
	// Create payload
	// txPayload = &payload.TransferAsset{}
	// Create attributes
	txAttr := tx.NewTxAttribute(tx.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	attributes := make([]*tx.TxAttribute, 0)
	attributes = append(attributes, &txAttr)
	// Create program
	var program = &pg.Program{redeemScript, nil}
	// Create transaction
	return &tx.Transaction{
		TxType:        txType,
		Payload:       txPayload,
		Attributes:    attributes,
		UTXOInputs:    inputs,
		BalanceInputs: []*tx.BalanceTxInput{},
		Outputs:       outputs,
		Programs:      []*pg.Program{program},
		LockTime:      wallet.CurrentHeight(QueryHeightCode) - 1,
	}
}
