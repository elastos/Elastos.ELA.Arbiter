package complain

import (
	"math"

	"Elastos.ELA.Arbiter/arbitration/arbitratorgroup"
	. "Elastos.ELA.Arbiter/common"
	. "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/crypto"
	"bytes"
	"errors"
)

type ComplainItemImpl struct {
	UserAddress      string
	GenesisBlockHash string
	TransactionHash  Uint256
	IsFromMainBlock  bool

	redeemScript []byte
	signedData   []byte
}

func (item *ComplainItemImpl) GetUserAddress() string {
	return item.UserAddress
}

func (item *ComplainItemImpl) GetGenesisBlockHash() string {
	return item.GenesisBlockHash
}

func (item *ComplainItemImpl) GetTransactionHash() Uint256 {
	return item.TransactionHash
}

func (item *ComplainItemImpl) GetIsFromMainBlock() bool {
	return item.IsFromMainBlock
}

func (item *ComplainItemImpl) Accepted() bool {
	return true
}

func (item *ComplainItemImpl) Serialize() ([]byte, error) {
	return nil, nil
}

func (item *ComplainItemImpl) Deserialize(content []byte) error {
	return nil
}

func (item *ComplainItemImpl) Verify() bool {
	if item.IsFromMainBlock {
		return item.verifyMainTransaction()
	} else {
		return item.verifySideTransaction()
	}
}

func (item *ComplainItemImpl) verifyMainTransaction() bool {
	//todo call SPV module to verify
	return true
}

func (item *ComplainItemImpl) verifySideTransaction() bool {
	return true
}

func (item *ComplainItemImpl) createRedeemScript() error {
	M := int(math.Ceil(float64(arbitratorgroup.ArbitratorGroupSingleton.GetArbitratorsCount()) * float64(2) / float64(3)))
	publicKeys := make([]*crypto.PublicKey, arbitratorgroup.ArbitratorGroupSingleton.GetArbitratorsCount())
	for _, arbitrator := range arbitratorgroup.ArbitratorGroupSingleton.GetAllArbitrators() {
		temp := &crypto.PublicKey{}
		temp.FromString(arbitrator)
		publicKeys = append(publicKeys, temp)
	}

	bytes, err := CreateMultiSignRedeemScript(M, publicKeys)
	if err != nil {
		return err
	}

	item.redeemScript = bytes
	return nil
}

func (item *ComplainItemImpl) SignItem(password []byte, arbitrator arbitratorgroup.Arbitrator) error {
	// Check if current user is a valid signer
	var signerIndex = -1
	programHashes, err := item.getMultiSignSigners()
	if err != nil {
		return err
	}
	userProgramHash := arbitrator.GetProgramHash()
	for i, programHash := range programHashes {
		if *userProgramHash == *programHash {
			signerIndex = i
			break
		}
	}
	if signerIndex == -1 {
		return errors.New("[Wallet], Invalid multi sign signer")
	}
	// Sign transaction
	signedData, err := arbitrator.Sign(password, item)
	if err != nil {
		return err
	}
	// Append signature
	err = item.appendSignature(signerIndex, signedData)
	if err != nil {
		return err
	}

	return nil
}

func (item *ComplainItemImpl) appendSignature(signerIndex int, signature []byte) error {
	// Create new signature
	newSign := append([]byte{}, byte(len(signature)))
	newSign = append(newSign, signature...)

	param := item.signedData

	// Check if is first signature
	if param == nil {
		param = []byte{}
	} else {
		// Check if singer already signed
		publicKeys, err := item.getMultiSignPublicKeys()
		if err != nil {
			return err
		}
		buf := new(bytes.Buffer)
		data, err := item.Serialize()
		if err != nil {
			return err
		}

		buf.Write(data)
		for i := 0; i < len(param); i += SignatureScriptLength {
			// Remove length byte
			sign := param[i : i+SignatureScriptLength][1:]
			publicKey := publicKeys[signerIndex][1:]
			pubKey, err := crypto.DecodePoint(publicKey)
			if err != nil {
				return err
			}
			err = crypto.Verify(*pubKey, buf.Bytes(), sign)
			if err == nil {
				return errors.New("signer already signed")
			}
		}
	}

	buf := new(bytes.Buffer)
	buf.Write(param)
	buf.Write(newSign)

	item.signedData = buf.Bytes()

	return nil
}

func (item *ComplainItemImpl) getMultiSignSigners() ([]*Uint168, error) {
	scripts, err := item.getMultiSignPublicKeys()
	if err != nil {
		return nil, err
	}

	var signers []*Uint168
	for _, script := range scripts {
		script = append(script, STANDARD)
		hash, _ := ToProgramHash(script)
		signers = append(signers, hash)
	}

	return signers, nil
}

func (item *ComplainItemImpl) getMultiSignPublicKeys() ([][]byte, error) {
	if len(item.redeemScript) < MinMultiSignCodeLength || item.redeemScript[len(item.redeemScript)-1] != MULTISIG {
		return nil, errors.New("not a valid multi sign transaction item.redeemScript, length not enough")
	}
	// remove last byte MULTISIG
	item.redeemScript = item.redeemScript[:len(item.redeemScript)-1]
	// remove m
	item.redeemScript = item.redeemScript[1:]
	// remove n
	item.redeemScript = item.redeemScript[:len(item.redeemScript)-1]
	if len(item.redeemScript)%(PublicKeyScriptLength-1) != 0 {
		return nil, errors.New("not a valid multi sign transaction item.redeemScript, length not match")
	}

	var publicKeys [][]byte
	i := 0
	for i < len(item.redeemScript) {
		script := make([]byte, PublicKeyScriptLength-1)
		copy(script, item.redeemScript[i:i+PublicKeyScriptLength-1])
		i += PublicKeyScriptLength - 1
		publicKeys = append(publicKeys, script)
	}
	return publicKeys, nil
}
