package cs

import (
	"io"

	"github.com/elastos/Elastos.ELA.Utility/common"
)

type TxCacheClearMessage struct {
	Command    string
	RemovedTxs []string
}

func (msg *TxCacheClearMessage) CMD() string {
	return msg.Command
}

func (msg *TxCacheClearMessage) Serialize(w io.Writer) error {
	common.WriteElement(w, len(msg.RemovedTxs))
	for _, str := range msg.RemovedTxs {
		common.WriteElement(w, []byte(str))
	}
	return nil
}

func (msg *TxCacheClearMessage) Deserialize(r io.Reader) error {
	var length int
	var strByte []byte
	common.ReadElement(r, length)
	msg.RemovedTxs = make([]string, length)
	for i := 0; i < length; i++ {
		common.ReadElement(r, strByte)
		msg.RemovedTxs[i] = string(strByte)
	}
	return nil
}
