package cs

import (
	"io"
	"math/big"

	"github.com/elastos/Elastos.ELA/common"
)

type SchnorrWithdrawRequestRProposalContent struct {
	Nonce []byte
	R     KRP
}

type KRP struct {
	K0 *big.Int
	Rx *big.Int
	Ry *big.Int
	Px *big.Int
	Py *big.Int
}

func (r *KRP) Serialize(w io.Writer) error {
	if err := common.WriteVarBytes(w, r.K0.Bytes()); err != nil {
		return err
	}
	if err := common.WriteVarBytes(w, r.Rx.Bytes()); err != nil {
		return err
	}
	if err := common.WriteVarBytes(w, r.Ry.Bytes()); err != nil {
		return err
	}
	if err := common.WriteVarBytes(w, r.Px.Bytes()); err != nil {
		return err
	}
	if err := common.WriteVarBytes(w, r.Py.Bytes()); err != nil {
		return err
	}
	return nil
}

func (c *KRP) Deserialize(r io.Reader) error {
	k0, err := common.ReadVarBytes(r, 64, "k0")
	if err != nil {
		return err
	}
	c.K0 = new(big.Int).SetBytes(k0)

	rx, err := common.ReadVarBytes(r, 64, "rx")
	if err != nil {
		return err
	}
	c.Rx = new(big.Int).SetBytes(rx)

	ry, err := common.ReadVarBytes(r, 64, "ry")
	if err != nil {
		return err
	}
	c.Ry = new(big.Int).SetBytes(ry)

	px, err := common.ReadVarBytes(r, 64, "px")
	if err != nil {
		return err
	}
	c.Px = new(big.Int).SetBytes(px)

	py, err := common.ReadVarBytes(r, 64, "py")
	if err != nil {
		return err
	}
	c.Py = new(big.Int).SetBytes(py)
	return nil
}

func (c *SchnorrWithdrawRequestRProposalContent) SerializeUnsigned(w io.Writer, feedback bool) error {
	if err := common.WriteVarBytes(w, c.Nonce); err != nil {
		return err
	}

	if feedback {
		if err := c.R.Serialize(w); err != nil {
			return err
		}
	}

	return nil
}

func (c *SchnorrWithdrawRequestRProposalContent) Serialize(w io.Writer, feedback bool) error {
	return c.SerializeUnsigned(w, feedback)
}

func (c *SchnorrWithdrawRequestRProposalContent) Deserialize(r io.Reader, feedback bool) (err error) {
	c.Nonce, err = common.ReadVarBytes(r, 64, "nonce")
	if err != nil {
		return err
	}

	if feedback {
		if err = c.R.Deserialize(r); err != nil {
			return err
		}
	}

	return
}

func (d *SchnorrWithdrawRequestRProposalContent) Hash() common.Uint256 {
	return common.Hash(d.Nonce)
}

func (d *SchnorrWithdrawRequestRProposalContent) Check(client interface{}) error {
	return nil
}
