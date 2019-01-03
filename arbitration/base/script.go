package base

import (
	"bytes"
	"errors"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
)

type OpCode byte

func CreateWithdrawRedeemScript(M int, publicKeys []*crypto.PublicKey) ([]byte, error) {
	return createMultiSignRedeemScriptInner(M, publicKeys, common.CROSSCHAIN)
}

func createMultiSignRedeemScriptInner(M int, publicKeys []*crypto.PublicKey, scriptType byte) ([]byte, error) {
	// Write M
	opCode := OpCode(byte(crypto.PUSH1) + byte(M) - 1)
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(opCode))

	//sort pubkey
	crypto.SortPublicKeys(publicKeys)

	// Write public keys
	for _, pubkey := range publicKeys {
		content, err := pubkey.EncodePoint(true)
		if err != nil {
			return nil, errors.New("create multi sign redeem script, encode public key failed")
		}
		buf.WriteByte(byte(len(content)))
		buf.Write(content)
	}

	// Write N
	N := len(publicKeys)
	opCode = OpCode(byte(crypto.PUSH1) + byte(N) - 1)
	buf.WriteByte(byte(opCode))
	buf.WriteByte(scriptType)

	return buf.Bytes(), nil
}
