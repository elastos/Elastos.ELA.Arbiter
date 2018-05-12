package base

import (
	"bytes"
	"errors"

	. "github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA.Utility/crypto"
	. "github.com/elastos/Elastos.ELA/core"
	"strings"
)

func PublicKeyFromString(str string) (*PublicKey, error) {
	keyBytes, err := HexStringToBytes(strings.TrimSpace(str))
	if err != nil {
		return nil, err
	}
	publicKey, err := DecodePoint(keyBytes)
	if err != nil {
		return nil, err
	}

	return publicKey, nil
}

func StandardAcccountPublicKeyToProgramHash(key *PublicKey) (*Uint168, error) {
	targetRedeemScript, err := CreateStandardRedeemScript(key)
	if err != nil {
		return nil, err
	}
	targetProgramHash, err := ToProgramHash(targetRedeemScript)
	if err != nil {
		return nil, err
	}
	return targetProgramHash, err
}

func MergeSignToTransaction(newSign []byte, signerIndex int, txn *Transaction) (int, error) {
	param := txn.Programs[0].Parameter

	// Check if is first signature
	if param == nil {
		param = []byte{}
	} else {
		// Check if singer already signed
		publicKeys, err := ParseMultisigScript(txn.Programs[0].Code)
		if err != nil {
			return 0, err
		}
		buf := new(bytes.Buffer)
		txn.Serialize(buf)
		for i := 0; i < len(param); i += SignatureScriptLength {
			// Remove length byte
			sign := param[i : i+SignatureScriptLength][1:]
			publicKey := publicKeys[signerIndex][1:]
			pubKey, err := DecodePoint(publicKey)
			if err != nil {
				return 0, err
			}
			err = Verify(*pubKey, buf.Bytes(), sign)
			if err == nil {
				return 0, errors.New("signer already signed")
			}
		}
	}

	buf := new(bytes.Buffer)
	buf.Write(param)
	buf.Write(newSign)

	txn.Programs[0].Parameter = buf.Bytes()
	return len(txn.Programs[0].Parameter) / (SignatureScriptLength - 1), nil
}

func SubstractTransactionHashes(hashSet, subSet []string) []string {
	var result []string

	for _, hash := range hashSet {
		if !hasHash(subSet, hash) {
			result = append(result, hash)
		}
	}
	return result
}

func hasHash(hashSet []string, hash string) bool {
	for _, item := range hashSet {
		if item == hash {
			return true
		}
	}
	return false
}
