package mainchain

import (
	"errors"

	. "Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "Elastos.ELA.Arbiter/common"
	. "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/crypto"
	"bytes"
)

type DistributedTransactionItem struct {
	TargetArbitratorPublicKey   *crypto.PublicKey
	TargetArbitratorProgramHash *Uint168
	RawTransaction              *Transaction

	redeemScript []byte
	signedData   []byte
}

func (item *DistributedTransactionItem) InitScript(arbitrator Arbitrator) error {
	err := item.createMultiSignRedeemScript(arbitrator)
	if err != nil {
		return err
	}

	return nil
}

func (item *DistributedTransactionItem) Sign(password []byte, arbitrator Arbitrator) error {
	// Check if current user is a valid signer
	var signerIndex = -1
	var targetIndex = -1
	programHashes, err := item.getMultiSignSigners()
	if err != nil {
		return err
	}
	if len(programHashes) != 2 {
		return errors.New("Invalid multi sign signer count.")
	}

	userProgramHash := arbitrator.GetProgramHash()
	for i, programHash := range programHashes {
		if *userProgramHash == *programHash {
			signerIndex = i
		} else if *item.TargetArbitratorProgramHash == *programHash {
			targetIndex = i
		}
	}
	if signerIndex == -1 || targetIndex == -1 {
		return errors.New("Invalid multi sign signer")
	}
	// Sign transaction
	buf := new(bytes.Buffer)
	err = item.RawTransaction.SerializeUnsigned(buf)
	if err != nil {
		return err
	}

	newSign, err := arbitrator.Sign(password, buf.Bytes())
	if err != nil {
		return err
	}
	// Append signature
	err = item.appendSignature(signerIndex, newSign, !arbitrator.IsOnDuty())
	if err != nil {
		return err
	}

	return nil
}

func (item *DistributedTransactionItem) GetSignedData() []byte {
	return item.signedData
}

func (item *DistributedTransactionItem) ParseFeedbackSignedData() ([]byte, error) {
	if len(item.signedData) != SignatureScriptLength*2 {
		return nil, errors.New("Invalid sign data.")
	}

	sign := item.signedData[SignatureScriptLength:][1:]

	buf := new(bytes.Buffer)
	err := item.RawTransaction.SerializeUnsigned(buf)
	if err != nil {
		return nil, err
	}

	err = crypto.Verify(*item.TargetArbitratorPublicKey, buf.Bytes(), sign)
	if err != nil {
		return nil, errors.New("Invalid sign data.")
	}

	return sign, nil
}

func (item *DistributedTransactionItem) Serialize() ([]byte, error) {
	return nil, nil
}

func (item *DistributedTransactionItem) Deserialize(content []byte) error {
	return nil
}

func (item *DistributedTransactionItem) createMultiSignRedeemScript(arbitrator Arbitrator) error {
	signers := make([]*crypto.PublicKey, 2)
	signers[0] = arbitrator.GetPublicKey()
	signers[1] = item.TargetArbitratorPublicKey

	script, err := CreateMultiSignRedeemScript(2, signers)
	if err != nil {
		return err
	}

	item.redeemScript = script
	return nil
}

func (item *DistributedTransactionItem) getMultiSignSigners() ([]*Uint168, error) {
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

func (item *DistributedTransactionItem) getMultiSignPublicKeys() ([][]byte, error) {
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

func (item *DistributedTransactionItem) appendSignature(signerIndex int, signature []byte, isFeedback bool) error {
	// Create new signature
	newSign := append([]byte{}, byte(len(signature)))
	newSign = append(newSign, signature...)

	signedData := item.signedData

	if !isFeedback {
		if signedData == nil {
			signedData = []byte{}
		} else {
			return errors.New("Can not sign a not-null trasaction.")
		}
	} else {
		if len(signedData) != SignatureScriptLength {
			return errors.New("Invalid sign data.")
		}

		sign := signedData[1:]
		currentArbitratorPk := item.TargetArbitratorPublicKey

		buf := new(bytes.Buffer)
		err := item.RawTransaction.SerializeUnsigned(buf)
		if err != nil {
			return err
		}

		err = crypto.Verify(*currentArbitratorPk, buf.Bytes(), sign)
		if err != nil {
			return errors.New("Can not sign without current arbitrator's signing.")
		}
	}

	buf := new(bytes.Buffer)
	buf.Write(signedData)
	buf.Write(newSign)

	item.signedData = buf.Bytes()

	return nil
}
