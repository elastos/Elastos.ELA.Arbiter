package cs

import (
	"io"

	"github.com/elastos/Elastos.ELA.Utility/common"
)

type SignMessage struct {
	Command string
	Content []byte
}

func (msg *SignMessage) CMD() string {
	return msg.Command
}

func (msg *SignMessage) Serialize(w io.Writer) error {
	return common.WriteVarBytes(w, msg.Content)
}

func (msg *SignMessage) Deserialize(r io.Reader) error {
	content, err := common.ReadVarBytes(r)
	if err != nil {
		return err
	}
	msg.Content = content
	return nil
}
