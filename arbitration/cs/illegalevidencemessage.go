package cs

import (
	"io"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA/p2p/msg"
)

type IllegalEvidenceMessage struct {
	base.SidechainIllegalData
}

func (i *IllegalEvidenceMessage) CMD() string {
	return IllegalEvidence
}

func (i *IllegalEvidenceMessage) MaxLength() uint32 {
	return msg.MaxBlockSize
}

func (i *IllegalEvidenceMessage) Serialize(w io.Writer) error {
	return i.SidechainIllegalData.Serialize(w)
}

func (i *IllegalEvidenceMessage) Deserialize(r io.Reader) error {
	return i.SidechainIllegalData.Deserialize(r)
}
