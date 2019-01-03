package cs

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/p2p/msg"
)

type SignMessage struct {
	Command string
	Content []byte
}

func (s *SignMessage) CMD() string {
	return s.Command
}

func (s *SignMessage) MaxLength() uint32 {
	return msg.MaxBlockSize
}

func (s *SignMessage) Serialize(w io.Writer) error {
	return common.WriteVarBytes(w, s.Content)
}

func (s *SignMessage) Deserialize(r io.Reader) error {
	content, err := common.ReadVarBytes(r, msg.MaxBlockSize, "Content")
	if err != nil {
		return err
	}
	s.Content = content
	return nil
}
