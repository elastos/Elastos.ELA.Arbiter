package crypto

import (
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"math/big"
)

var (
	// Curve is a KoblitzCurve which implements secp256r1.
	Curve = elliptic.P256()
	// One holds a big integer of 1
	One = new(big.Int).SetInt64(1)
	// Two holds a big integer of 2
	Two = new(big.Int).SetInt64(2)
	// The order of the base point
	N = Curve.Params().N
	// The order of the underlying field
	P = Curve.Params().P
	// The constant of the curve equation
	B = Curve.Params().B
	// The size of the underlying field
	BitSize = Curve.Params().BitSize
)

func getE(Px, Py *big.Int, rX []byte, m []byte) *big.Int {
	r := append(rX, Marshal(Px, Py)...)
	r = append(r, m[:]...)
	h := sha256.Sum256(r)
	i := new(big.Int).SetBytes(h[:])
	return i.Mod(i, N)
}

func IntToByte(i *big.Int) []byte {
	b1, b2 := [32]byte{}, i.Bytes()
	copy(b1[32-len(b2):], b2)
	return b1[:]
}

func Marshal(x, y *big.Int) []byte {
	byteLen := (BitSize + 7) >> 3

	ret := make([]byte, 1+byteLen)
	ret[0] = 2

	xBytes := x.Bytes()
	copy(ret[1+byteLen-len(xBytes):], xBytes)
	ret[0] += byte(y.Bit(0))
	return ret
}

func deterministicGetK0(d []byte) (*big.Int, error) {
	for {
		message, err := randomBytes(32)
		if err != nil {
			return nil, errors.New("random bytes error:" + err.Error())
		}
		h := sha256.Sum256(append(d, message[:]...))
		i := new(big.Int).SetBytes(h[:])
		k0 := i.Mod(i, N)
		if k0.Sign() == 0 {
			return nil, errors.New("k0 is zero")
		}
		return k0, nil
	}
}

func randomBytes(len int) (data []byte, err error) {
	data = make([]byte, len)
	_, err = rand.Read(data)
	return
}

// GetR calcaulate k0 rx ry px and py.
func GetR(privateKey *big.Int) (k0 *big.Int, rx *big.Int, ry *big.Int, px *big.Int, py *big.Int, err error) {
	log.Info("################ privateKey:", privateKey)
	if privateKey.Cmp(One) < 0 || privateKey.Cmp(new(big.Int).Sub(N, One)) > 0 {
		err = errors.New("the private key must be an integer in the range 1..n-1")
		return
	}

	d := IntToByte(privateKey)
	k0, err = deterministicGetK0(d)
	if err != nil {
		return
	}

	rx, ry = Curve.ScalarBaseMult(IntToByte(k0))
	px, py = Curve.ScalarBaseMult(d)
	log.Info("################ px:", *px)

	return
}

func GetE(rxs []*big.Int, rys []*big.Int, pxs []*big.Int, pys []*big.Int, message []byte) *big.Int {
	Px, Py := new(big.Int), new(big.Int)
	Rx, Ry := new(big.Int), new(big.Int)
	for i, _ := range rxs {
		Rx, Ry = Curve.Add(Rx, Ry, rxs[i], rys[i])
		Px, Py = Curve.Add(Px, Py, pxs[i], pys[i])
	}
	rX := IntToByte(Rx)
	return getE(Px, Py, rX, message[:])
}

func GetEMulPrivateKey(privateKeys *big.Int, e *big.Int) *big.Int {
	return new(big.Int).Mul(e, privateKeys)
}

func GetK(Ry, k0 *big.Int) *big.Int {
	if big.Jacobi(Ry, P) == 1 {
		return k0
	}
	return k0.Sub(N, k0)
}

func GetS(Rx *big.Int, s *big.Int) [64]byte {
	var signature [64]byte
	copy(signature[:32], IntToByte(Rx))
	copy(signature[32:], IntToByte(s.Mod(s, Curve.Params().N)))
	return signature
}
