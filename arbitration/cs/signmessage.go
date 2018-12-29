package cs

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/p2p/msg"
)

type DistributedItemMessage struct {
	Content []byte
}

func (s *DistributedItemMessage) CMD() string {
	return DistributeItemCommand
}

func (s *DistributedItemMessage) MaxLength() uint32 {
	return msg.MaxBlockSize
}

func (s *DistributedItemMessage) Serialize(w io.Writer) error {
	return common.WriteVarBytes(w, s.Content)
}

func (s *DistributedItemMessage) Deserialize(r io.Reader) error {
	content, err := common.ReadVarBytes(r, msg.MaxBlockSize, "Content")
	if err != nil {
		return err
	}
	s.Content = content
	return nil
}
