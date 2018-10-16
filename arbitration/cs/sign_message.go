package cs

import (
	"io"

	"github.com/elastos/Elastos.ELA.Utility/common"
)

const MaxSignMessageDataSize = 1000

type SignMessage struct {
	Command string
	Content []byte
}

func (msg *SignMessage) CMD() string {
	return msg.Command
}

func (msg *SignMessage) MaxLength() uint32 {
	return MaxSignMessageDataSize
}

func (msg *SignMessage) Serialize(w io.Writer) error {
	return common.WriteVarBytes(w, msg.Content)
}

func (msg *SignMessage) Deserialize(r io.Reader) error {
	content, err := common.ReadVarBytes(r, MaxSignMessageDataSize, "Content")
	if err != nil {
		return err
	}
	msg.Content = content
	return nil
}
