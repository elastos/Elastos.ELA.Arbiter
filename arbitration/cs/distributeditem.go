package cs

import (
	"bytes"
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/crypto"
)

type TransactionType byte

const (
	WithdrawTransaction      TransactionType = 0x00
	IllegalTransaction       TransactionType = 0x01
	ReturnDepositTransaction TransactionType = 0x02
)

type DistributeContentType byte

const (
	MultisigContent         DistributeContentType = 0x00
	IllegalContent          DistributeContentType = 0x01
	SchnorrMultisigContent2 DistributeContentType = 0x02
	SchnorrMultisigContent3 DistributeContentType = 0x03

	AnswerMultisigContent         DistributeContentType = 0x10
	AnswerIllegalContent          DistributeContentType = 0x11
	AnswerSchnorrMultisigContent2 DistributeContentType = 0x12
	AnswerSchnorrMultisigContent3 DistributeContentType = 0x13
)

const MaxRedeemScriptDataSize = 10000

type DistributedItem struct {
	TargetArbitratorPublicKey      *crypto.PublicKey
	TargetArbitratorProgramHash    *common.Uint168
	TransactionType                TransactionType
	Type                           DistributeContentType
	ItemContent                    base.DistributedContent
	SchnorrRequestRProposalContent SchnorrWithdrawRequestRProposalContent
	SchnorrRequestSProposalContent SchnorrWithdrawRequestSProposalContent

	redeemScript []byte
	signedData   []byte
}

type DistrubutedItemFunc interface {
	GetArbitratorGroupInfoByHeight(height uint32) (*rpc.ArbitratorGroupInfo, error)
	GetCurrentHeight() (uint32, error)
}

type DistrubutedItemFuncImpl struct {
}

func (item *DistributedItem) InitScript(arbitrator arbitrator.Arbitrator) error {
	err := item.createMultiSignRedeemScript()
	if err != nil {
		return err
	}

	return nil
}

func (item *DistributedItem) GetRedeemScript() []byte {
	return item.redeemScript
}

func (item *DistributedItem) SetRedeemScript(script []byte) {
	item.redeemScript = script
}

func (item *DistributedItem) Sign(arbitrator arbitrator.Arbitrator, isFeedback bool, itemFunc DistrubutedItemFunc) error {
	// Check if current user is a valid signer
	var signerIndex = -1
	programHashes, err := item.getMultiSignSigners()
	if err != nil {
		return err
	}

	pkBuf, err := arbitrator.GetPublicKey().EncodePoint(true)
	if err != nil {
		return err
	}

	userProgramHash, err := contract.PublicKeyToStandardProgramHash(pkBuf)
	if err != nil {
		return err
	}
	for i, programHash := range programHashes {
		if userProgramHash.IsEqual(*programHash) {
			signerIndex = i
		}
	}
	if signerIndex == -1 {
		return errors.New("Invalid multi sign signer")
	}
	// Sign transaction
	buf := new(bytes.Buffer)
	err = item.ItemContent.SerializeUnsigned(buf)
	if err != nil {
		return err
	}

	newSign, err := arbitrator.Sign(buf.Bytes())
	if err != nil {
		return err
	}
	// Append signature
	err = item.appendSignature(signerIndex, newSign, isFeedback, itemFunc)
	if err != nil {
		return err
	}

	return nil
}

func (item *DistributedItem) SchnorrSign2(arbitrator arbitrator.Arbitrator) error {
	// Sign transaction
	buf := new(bytes.Buffer)
	err := item.Serialize(buf)
	if err != nil {
		return err
	}

	newSign, err := arbitrator.Sign(buf.Bytes())
	if err != nil {
		return err
	}
	// Record signature
	item.signedData = newSign
	return nil
}

func (item *DistributedItem) GetSignedData() []byte {
	return item.signedData
}

func (item *DistributedItem) ParseFeedbackSignedData() ([]byte, string, error) {
	log.Info("item.signedData length ", len(item.signedData))
	if len(item.signedData) != crypto.SignatureScriptLength*2 {
		return nil, "ParseFeedbackSignedData invalid sign data length.", nil
	}

	sign := item.signedData[crypto.SignatureScriptLength:]

	buf := new(bytes.Buffer)
	err := item.ItemContent.SerializeUnsigned(buf)
	if err != nil {
		return nil, "", err
	}

	err = crypto.Verify(*item.TargetArbitratorPublicKey, buf.Bytes(), sign[1:])
	if err != nil {
		return nil, "", errors.New("ParseFeedbackSignedData invalid sign data.")
	}

	return sign, "", nil
}

func (item *DistributedItem) CheckSchnorrFeedbackRequestRSignedData() error {
	if len(item.signedData) == 0 {
		return errors.New("CheckSchnorrFeedbackRequestRSignedData invalid sign data length.")
	}

	buf := new(bytes.Buffer)
	err := item.Serialize(buf)
	if err != nil {
		return err
	}

	err = crypto.Verify(*item.TargetArbitratorPublicKey, buf.Bytes(), item.signedData[1:])
	if err != nil {
		return errors.New("CheckSchnorrFeedbackProposalSignedData invalid sign data.")
	}

	return nil
}

func (item *DistributedItem) Serialize(w io.Writer) error {
	publickeyBytes, _ := item.TargetArbitratorPublicKey.EncodePoint(true)
	if err := common.WriteVarBytes(w, publickeyBytes); err != nil {
		return errors.New("TargetArbitratorPublicKey serialization failed.")
	}
	if err := item.TargetArbitratorProgramHash.Serialize(w); err != nil {
		return errors.New("TargetArbitratorProgramHash serialization failed.")
	}
	if err := common.WriteUint8(w, byte(item.TransactionType)); err != nil {
		return err
	}
	if err := common.WriteUint8(w, byte(item.Type)); err != nil {
		return err
	}
	switch item.Type {
	case MultisigContent, AnswerMultisigContent:
		if err := item.ItemContent.Serialize(w); err != nil {
			return err
		}
		if err := common.WriteVarBytes(w, item.redeemScript); err != nil {
			return errors.New("redeemScript serialization failed.")
		}
		if err := common.WriteVarBytes(w, item.signedData); err != nil {
			return errors.New("signedData serialization failed.")
		}
	case IllegalContent, AnswerIllegalContent:
		if err := common.WriteVarBytes(w, item.redeemScript); err != nil {
			return errors.New("redeemScript serialization failed.")
		}
		if err := common.WriteVarBytes(w, item.signedData); err != nil {
			return errors.New("signedData serialization failed.")
		}
	case SchnorrMultisigContent2:
		if err := item.SchnorrRequestRProposalContent.Serialize(w, false); err != nil {
			return err
		}
	case AnswerSchnorrMultisigContent2:
		if err := item.SchnorrRequestRProposalContent.Serialize(w, true); err != nil {
			return err
		}
	case SchnorrMultisigContent3:
		if err := item.SchnorrRequestSProposalContent.Serialize(w, false); err != nil {
			return err
		}
	case AnswerSchnorrMultisigContent3:
		if err := item.SchnorrRequestSProposalContent.Serialize(w, true); err != nil {
			return err
		}
	}

	return nil
}

func (item *DistributedItem) Deserialize(r io.Reader) error {
	publickeyBytes, err := common.ReadVarBytes(r, crypto.PublicKeyScriptLength, "publickey bytes")
	if err != nil {
		return errors.New("TargetArbitratorPublicKey deserialization failed.")
	}
	publickey, _ := crypto.DecodePoint(publickeyBytes)
	item.TargetArbitratorPublicKey = publickey

	item.TargetArbitratorProgramHash = nil
	item.TargetArbitratorProgramHash = new(common.Uint168)
	if err = item.TargetArbitratorProgramHash.Deserialize(r); err != nil {
		return errors.New("TargetArbitratorProgramHash deserialization failed.")
	}

	transactionType, err := common.ReadUint8(r)
	if err != nil {
		return err
	}
	item.TransactionType = TransactionType(transactionType)

	contentType, err := common.ReadUint8(r)
	if err != nil {
		return err
	}
	item.Type = DistributeContentType(contentType)

	switch item.Type {
	case MultisigContent, AnswerMultisigContent:
		item.ItemContent = &TxDistributedContent{Tx: new(types.Transaction)}
		if err = item.ItemContent.Deserialize(r); err != nil {
			return errors.New("ItemContent deserialization failed." + err.Error())
		}
	case IllegalContent, AnswerIllegalContent:
	case SchnorrMultisigContent2:
		if err = item.SchnorrRequestRProposalContent.Deserialize(r, false); err != nil {
			return errors.New("SchnorrRequestRProposalContent deserialization failed." + err.Error())
		}
	case AnswerSchnorrMultisigContent2:
		if err = item.SchnorrRequestRProposalContent.Deserialize(r, true); err != nil {
			return errors.New("Answer SchnorrRequestRProposalContent deserialization failed." + err.Error())
		}
	case SchnorrMultisigContent3:
		if err = item.SchnorrRequestSProposalContent.Deserialize(r, true); err != nil {
			return errors.New("Answer SchnorrRequestSProposalContent deserialization failed." + err.Error())
		}
	case AnswerSchnorrMultisigContent3:
		if err = item.SchnorrRequestSProposalContent.Deserialize(r, true); err != nil {
			return errors.New("Answer SchnorrRequestSProposalContent deserialization failed." + err.Error())
		}
	}

	redeemScript, err := common.ReadVarBytes(r, MaxRedeemScriptDataSize, "redeem script")
	if err != nil {
		return errors.New("redeemScript deserialization failed.")
	}
	item.redeemScript = redeemScript

	signedData, err := common.ReadVarBytes(r, crypto.SignatureScriptLength*2, "signed data")
	if err != nil {
		return errors.New("signedData deserialization failed.")
	}
	item.signedData = signedData

	return nil
}

func (item *DistributedItem) createMultiSignRedeemScript() error {
	script, err := CreateRedeemScript()
	if err != nil {
		return err
	}

	item.SetRedeemScript(script)
	return nil
}

func (item *DistributedItem) getMultiSignSigners() ([]*common.Uint168, error) {
	scripts, err := item.getMultiSignPublicKeys()
	if err != nil {
		return nil, err
	}

	var signers []*common.Uint168
	for _, script := range scripts {
		hash, _ := contract.PublicKeyToStandardProgramHash(script[1:])
		signers = append(signers, hash)
	}

	return signers, nil
}

func (item *DistributedItem) getMultiSignPublicKeys() ([][]byte, error) {
	if len(item.redeemScript) < crypto.MinMultiSignCodeLength || item.redeemScript[len(item.redeemScript)-1] != common.CROSSCHAIN {
		return nil, errors.New("not a valid multi sign transaction item.redeemScript, length not enough")
	}

	redeemScript := item.redeemScript
	// remove last byte MULTISIG
	redeemScript = redeemScript[:len(redeemScript)-1]
	// remove m
	redeemScript = redeemScript[1:]
	// remove n
	redeemScript = redeemScript[:len(redeemScript)-1]
	if len(redeemScript)%(crypto.PublicKeyScriptLength-1) != 0 {
		return nil, errors.New("not a valid multi sign transaction item.redeemScript, length not match")
	}

	var publicKeys [][]byte
	i := 0
	for i < len(redeemScript) {
		script := make([]byte, crypto.PublicKeyScriptLength-1)
		copy(script, redeemScript[i:i+crypto.PublicKeyScriptLength-1])
		i += crypto.PublicKeyScriptLength - 1
		publicKeys = append(publicKeys, script)
	}
	return publicKeys, nil
}

func (item *DistributedItem) IsFeedback() bool {
	return len(item.signedData)/crypto.SignatureScriptLength == 2
}

func (itemFunc *DistrubutedItemFuncImpl) GetArbitratorGroupInfoByHeight(height uint32) (*rpc.ArbitratorGroupInfo, error) {
	return rpc.GetArbitratorGroupInfoByHeight(height)
}

func (itemFunc *DistrubutedItemFuncImpl) GetCurrentHeight() (uint32, error) {
	return rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
}

func (item *DistributedItem) appendSignature(signerIndex int, signature []byte, isFeedback bool, itemFunc DistrubutedItemFunc) error {
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
		if len(signedData) != crypto.SignatureScriptLength {
			return errors.New("Invalid sign data.")
		}

		sign := signedData[1:]
		targetPk := item.TargetArbitratorPublicKey

		blockHeight, err := itemFunc.GetCurrentHeight()
		if err != nil {
			return err
		}
		groupInfo, err := itemFunc.GetArbitratorGroupInfoByHeight(blockHeight)
		if err != nil {
			return err
		}

		onDutyArbitratorPk, err :=
			base.PublicKeyFromString(groupInfo.Arbitrators[groupInfo.OnDutyArbitratorIndex])
		if err != nil {
			return err
		}

		if !crypto.Equal(targetPk, onDutyArbitratorPk) {
			return errors.New("Can not sign without current arbitrator's signing.")
		}

		buf := new(bytes.Buffer)
		err = item.ItemContent.SerializeUnsigned(buf)
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
