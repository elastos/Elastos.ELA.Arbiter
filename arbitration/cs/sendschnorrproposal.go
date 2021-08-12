package cs

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/elanet/pact"
)

type SendSchnorrProposalMessage struct {
	NonceHash common.Uint256
}

func (s *SendSchnorrProposalMessage) CMD() string {
	return SendSchnorrItemommand
}

func (s *SendSchnorrProposalMessage) MaxLength() uint32 {
	return pact.MaxBlockContextSize
}

func (s *SendSchnorrProposalMessage) Serialize(w io.Writer) error {
	return s.NonceHash.Serialize(w)
}

func (s *SendSchnorrProposalMessage) Deserialize(r io.Reader) error {
	return s.NonceHash.Deserialize(r)
}
