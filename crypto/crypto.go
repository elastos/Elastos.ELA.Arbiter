package crypto

import (
	"fmt"
	"errors"
	"math/big"
	"crypto/rand"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/elliptic"
)

const (
	SIGNRLEN     = 32
	SIGNATURELEN = 64
)

type CryptoAlgSet struct {
	EccParams elliptic.CurveParams
	Curve     elliptic.Curve
}

var algSet CryptoAlgSet

type PublicKey struct {
	X, Y *big.Int
}

func init() {
	algSet.Curve = elliptic.P256()
	algSet.EccParams = *(algSet.Curve.Params())
}

func GenKeyPair() ([]byte, *PublicKey, error) {

	privateKey, err := ecdsa.GenerateKey(algSet.Curve, rand.Reader)
	if err != nil {
		return nil, nil, errors.New("Generate key pair error")
	}

	publicKey := new(PublicKey)
	publicKey.X = new(big.Int).Set(privateKey.PublicKey.X)
	publicKey.Y = new(big.Int).Set(privateKey.PublicKey.Y)

	return privateKey.D.Bytes(), publicKey, nil
}

func Sign(priKey []byte, data []byte) ([]byte, error) {

	digest := sha256.Sum256(data)

	privateKey := new(ecdsa.PrivateKey)
	privateKey.Curve = algSet.Curve
	privateKey.D = big.NewInt(0)
	privateKey.D.SetBytes(priKey)

	r := big.NewInt(0)
	s := big.NewInt(0)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest[:])
	if err != nil {
		fmt.Printf("Sign error\n")
		return nil, err
	}

	signature := make([]byte, SIGNATURELEN)

	lenR := len(r.Bytes())
	lenS := len(s.Bytes())
	copy(signature[SIGNRLEN-lenR:], r.Bytes())
	copy(signature[SIGNATURELEN-lenS:], s.Bytes())
	return signature, nil
}

func Verify(publicKey PublicKey, data []byte, signature []byte) error {
	len := len(signature)
	if len != SIGNATURELEN {
		fmt.Printf("Unknown signature length %d\n", len)
		return errors.New("Unknown signature length")
	}

	r := new(big.Int).SetBytes(signature[:len/2])
	s := new(big.Int).SetBytes(signature[len/2:])

	digest := sha256.Sum256(data)

	pub := new(ecdsa.PublicKey)
	pub.Curve = algSet.Curve

	pub.X = new(big.Int).Set(publicKey.X)
	pub.Y = new(big.Int).Set(publicKey.Y)

	if ecdsa.Verify(pub, digest[:], r, s) {
		return nil
	} else {
		return errors.New("[Validation], Verify failed.")
	}

}

type PubKeySlice []*PublicKey

func (p PubKeySlice) Len() int { return len(p) }
func (p PubKeySlice) Less(i, j int) bool {
	r := p[i].X.Cmp(p[j].X)
	if r <= 0 {
		return true
	}
	return false
}
func (p PubKeySlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func Equal(e1 *PublicKey, e2 *PublicKey) bool {
	r := e1.X.Cmp(e2.X)
	if r != 0 {
		return false
	}
	r = e1.Y.Cmp(e2.Y)
	if r == 0 {
		return true
	}
	return false
}
