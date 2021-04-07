package cs

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"io"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
)

type DistributeContentType byte

const (
	MaxRedeemScriptDataSize = 10000

	TxDistribute      DistributeContentType = 0x00
	IllegalDistribute DistributeContentType = 0x01
	IllegalDepositTxs DistributeContentType = 0x02
)

type DistributedItem struct {
	TargetArbitratorPublicKey   *crypto.PublicKey
	TargetArbitratorProgramHash *common.Uint168
	Type                        DistributeContentType
	ItemContent                 base.DistributedContent

	redeemScript []byte
	signedData   []byte
}

type DistrubutedItemFunc interface {
	GetArbitratorGroupInfoByHeight(height uint32) (*rpc.ArbitratorGroupInfo, error)
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

func (item *DistributedItem) Serialize(w io.Writer) error {
	publickeyBytes, _ := item.TargetArbitratorPublicKey.EncodePoint(true)
	if err := common.WriteVarBytes(w, publickeyBytes); err != nil {
		return errors.New("TargetArbitratorPublicKey serialization failed.")
	}
	if err := item.TargetArbitratorProgramHash.Serialize(w); err != nil {
		return errors.New("TargetArbitratorProgramHash serialization failed.")
	}
	if err := common.WriteUint8(w, byte(item.Type)); err != nil {
		return err
	}
	if err := item.ItemContent.Serialize(w); err != nil {
		return err
	}
	if err := common.WriteVarBytes(w, item.redeemScript); err != nil {
		return errors.New("redeemScript serialization failed.")
	}
	if err := common.WriteVarBytes(w, item.signedData); err != nil {
		return errors.New("signedData serialization failed.")
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

	contentType, err := common.ReadUint8(r)
	if err != nil {
		return err
	}
	item.Type = DistributeContentType(contentType)

	switch item.Type {
	case TxDistribute:
		item.ItemContent = &TxDistributedContent{Tx: new(types.Transaction)}
		if err = item.ItemContent.Deserialize(r); err != nil {
			return errors.New("RawTransaction deserialization failed." + err.Error())
		}
	case IllegalDepositTxs:
		item.ItemContent = &IllegalDepositTx{DepositTxs: new(payload.IllegalDepositTxs)}
		if err = item.ItemContent.Deserialize(r); err != nil {
			return errors.New("RawTransaction deserialization failed." + err.Error())
		}
	case IllegalDistribute:

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

func (item *DistributedItem) appendSignature(signerIndex int, signature []byte, isFeedback bool, itemFunc DistrubutedItemFunc) error {
	// Create new signature
	newSign := append([]byte{}, byte(len(signature)))
	newSign = append(newSign, signature...)
	log.Info("appendSignature newSign ", len(newSign))
	signedData := item.signedData
	log.Info("appendSignature signedData ", len(item.signedData))
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

		blockHeight, err := item.ItemContent.CurrentBlockHeight()
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
			tarP, _ := targetPk.EncodePoint(true)
			onP, _ := onDutyArbitratorPk.EncodePoint(true)
			return errors.New("Can not sign without current arbitrator's signing. onduty arbiter is not the targetPk , actual pubkey " + hex.EncodeToString(tarP) + " onduty publicKey " + hex.EncodeToString(onP))
		}

		buf := new(bytes.Buffer)
		err = item.ItemContent.SerializeUnsigned(buf)
		if err != nil {
			return err
		}

		err = crypto.Verify(*targetPk, buf.Bytes(), sign)
		if err != nil {
			return errors.New("Can not sign without current arbitrator's signing." + err.Error())
		}
	}

	buf := new(bytes.Buffer)
	buf.Write(signedData)
	buf.Write(newSign)

	item.signedData = buf.Bytes()
	log.Info("appendSignature merge  ", item.signedData)
	return nil
}
