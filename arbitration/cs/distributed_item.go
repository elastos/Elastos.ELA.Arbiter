package cs

import (
	"errors"

	. "Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/common/serialization"
	. "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/crypto"
	"bytes"
	"io"
)

type DistributedItem struct {
	TargetArbitratorPublicKey   *crypto.PublicKey
	TargetArbitratorProgramHash *Uint168
	ItemContent                 *Transaction

	redeemScript []byte
	signedData   []byte
}

func (item *DistributedItem) InitScript(arbitrator Arbitrator) error {
	err := item.createMultiSignRedeemScript()
	if err != nil {
		return err
	}

	return nil
}

func (item *DistributedItem) Sign(arbitrator Arbitrator) error {
	// Check if current user is a valid signer
	var signerIndex = -1
	programHashes, err := item.getMultiSignSigners()
	if err != nil {
		return err
	}

	userProgramBytes, err := CreateStandardRedeemScript(arbitrator.GetPublicKey())
	if err != nil {
		return err
	}
	userProgramHash, err := Uint168FromBytes(userProgramBytes)
	if err != nil {
		return err
	}
	for i, programHash := range programHashes {
		if *userProgramHash == *programHash {
			signerIndex = i
		}
	}
	if signerIndex == -1 {
		return errors.New("Invalid multi sign signer")
	}
	// Sign transaction
	buf := new(bytes.Buffer)
	err = item.ItemContent.Serialize(buf)
	if err != nil {
		return err
	}

	newSign, err := arbitrator.Sign(buf.Bytes())
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

func (item *DistributedItem) GetSignedData() []byte {
	return item.signedData
}

func (item *DistributedItem) ParseFeedbackSignedData() ([]byte, error) {
	if len(item.signedData) != SignatureScriptLength*2 {
		return nil, errors.New("Invalid sign data.")
	}

	sign := item.signedData[SignatureScriptLength:][1:]

	buf := new(bytes.Buffer)
	err := item.ItemContent.Serialize(buf)
	if err != nil {
		return nil, err
	}

	err = crypto.Verify(*item.TargetArbitratorPublicKey, buf.Bytes(), sign)
	if err != nil {
		return nil, errors.New("Invalid sign data.")
	}

	return sign, nil
}

func (item *DistributedItem) Serialize(w io.Writer) error {
	publickeyBytes, _ := item.TargetArbitratorPublicKey.EncodePoint(true)
	if err := serialization.WriteVarBytes(w, publickeyBytes); err != nil {
		return errors.New("TargetArbitratorPublicKey serialization failed.")
	}
	if _, err := item.TargetArbitratorProgramHash.Serialize(w); err != nil {
		return errors.New("TargetArbitratorProgramHash serialization failed.")
	}
	if err := item.ItemContent.SerializeUnsigned(w); err != nil {
		return err
	}
	if err := serialization.WriteVarBytes(w, item.redeemScript); err != nil {
		return errors.New("redeemScript serialization failed.")
	}
	if err := serialization.WriteVarBytes(w, item.signedData); err != nil {
		return errors.New("signedData serialization failed.")
	}

	return nil
}

func (item *DistributedItem) Deserialize(r io.Reader) error {
	publickeyBytes, err := serialization.ReadVarBytes(r)
	if err != nil {
		return errors.New("TargetArbitratorPublicKey deserialization failed.")
	}
	publickey, _ := crypto.DecodePoint(publickeyBytes)
	item.TargetArbitratorPublicKey = publickey

	if err = item.TargetArbitratorProgramHash.Deserialize(r); err != nil {
		return errors.New("TargetArbitratorProgramHash deserialization failed.")
	}

	if err = item.ItemContent.DeserializeUnsigned(r); err != nil {
		return errors.New("RawTransaction deserialization failed.")
	}

	redeemScript, err := serialization.ReadVarBytes(r)
	if err != nil {
		return errors.New("redeemScript deserialization failed.")
	}
	item.redeemScript = redeemScript

	signedData, err := serialization.ReadVarBytes(r)
	if err != nil {
		return errors.New("signedData deserialization failed.")
	}
	item.signedData = signedData

	return nil
}

func (item *DistributedItem) isForComplain() bool {
	//todo judge if raw transaction is for complain (by payload)
	return false
}

func (item *DistributedItem) createMultiSignRedeemScript() error {
	script, err := CreateRedeemScript()
	if err != nil {
		return err
	}

	item.redeemScript = script
	return nil
}

func (item *DistributedItem) getMultiSignSigners() ([]*Uint168, error) {
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

func (item *DistributedItem) getMultiSignPublicKeys() ([][]byte, error) {
	if len(item.redeemScript) < MinMultiSignCodeLength || item.redeemScript[len(item.redeemScript)-1] != MULTISIG {
		return nil, errors.New("not a valid multi sign transaction item.redeemScript, length not enough")
	}

	redeemScript := item.redeemScript
	// remove last byte MULTISIG
	redeemScript = redeemScript[:len(redeemScript)-1]
	// remove m
	redeemScript = redeemScript[1:]
	// remove n
	redeemScript = redeemScript[:len(redeemScript)-1]
	if len(redeemScript)%(PublicKeyScriptLength-1) != 0 {
		return nil, errors.New("not a valid multi sign transaction item.redeemScript, length not match")
	}

	var publicKeys [][]byte
	i := 0
	for i < len(redeemScript) {
		script := make([]byte, PublicKeyScriptLength-1)
		copy(script, redeemScript[i:i+PublicKeyScriptLength-1])
		i += PublicKeyScriptLength - 1
		publicKeys = append(publicKeys, script)
	}
	return publicKeys, nil
}

func (item *DistributedItem) IsFeedback() bool {
	return len(item.signedData)/SignatureScriptLength == 2
}

func (item *DistributedItem) appendSignature(signerIndex int, signature []byte, isFeedback bool) error {
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
		targetPk := item.TargetArbitratorPublicKey

		if !item.isForComplain() {
			onDutyArbitratorPk := &crypto.PublicKey{}
			if err := onDutyArbitratorPk.FromString(ArbitratorGroupSingleton.GetOnDutyArbitrator()); err != nil {
				return err
			}
			if !crypto.Equal(targetPk, onDutyArbitratorPk) {
				return errors.New("Can not sign without current arbitrator's signing.")
			}
		}

		buf := new(bytes.Buffer)
		err := item.ItemContent.Serialize(buf)
		if err != nil {
			return err
		}

		err = crypto.Verify(*targetPk, buf.Bytes(), sign)
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
