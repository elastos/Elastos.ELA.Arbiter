package cs

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

type SchnorrWithdrawProposalContent struct {
	Nonce []byte
}

func (c *SchnorrWithdrawProposalContent) SerializeUnsigned(w io.Writer) error {
	return common.WriteVarBytes(w, c.Nonce)
}

func (c *SchnorrWithdrawProposalContent) Serialize(w io.Writer) error {
	return c.SerializeUnsigned(w)
}

func (c *SchnorrWithdrawProposalContent) Deserialize(r io.Reader) (err error) {
	c.Nonce, err = common.ReadVarBytes(r, 64, "nonce")
	return err
}

func (d *SchnorrWithdrawProposalContent) Hash() common.Uint256 {
	return common.Hash(d.Nonce)
}

func (d *SchnorrWithdrawProposalContent) Check(client interface{}) error {
	return nil
}
