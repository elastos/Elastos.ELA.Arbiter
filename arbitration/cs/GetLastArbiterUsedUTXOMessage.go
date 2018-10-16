package cs

import (
	"io"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/core"
)

const (
	MaxGetUsedUTXOMessageDataSize  = 1000
	MaxSendUsedUTXOMessageDataSize = 80000
)

type GetLastArbiterUsedUTXOMessage struct {
	Command        string
	GenesisAddress string
	Height         uint32
	Nonce          string
}

func (msg *GetLastArbiterUsedUTXOMessage) CMD() string {
	return msg.Command
}

func (msg *GetLastArbiterUsedUTXOMessage) MaxLength() uint32 {
	return MaxGetUsedUTXOMessageDataSize
}

func (msg *GetLastArbiterUsedUTXOMessage) Serialize(w io.Writer) error {
	err := common.WriteVarString(w, msg.GenesisAddress)
	if err != nil {
		return err
	}
	err = common.WriteUint32(w, msg.Height)
	if err != nil {
		return err
	}
	err = common.WriteVarString(w, msg.Nonce)
	if err != nil {
		return err
	}
	return nil
}

func (msg *GetLastArbiterUsedUTXOMessage) Deserialize(r io.Reader) error {
	genesisAddress, err := common.ReadVarString(r)
	if err != nil {
		return err
	}
	msg.GenesisAddress = genesisAddress
	height, err := common.ReadUint32(r)
	if err != nil {
		return err
	}
	msg.Height = height
	nonce, err := common.ReadVarString(r)
	if err != nil {
		return err
	}
	msg.Nonce = nonce
	return nil
}

type SendLastArbiterUsedUTXOMessage struct {
	Command        string
	GenesisAddress string
	Height         uint32
	OutPoints      []core.OutPoint
	Nonce          string
}

func (msg *SendLastArbiterUsedUTXOMessage) CMD() string {
	return msg.Command
}

func (msg *SendLastArbiterUsedUTXOMessage) MaxLength() uint32 {
	return MaxSendUsedUTXOMessageDataSize
}

func (msg *SendLastArbiterUsedUTXOMessage) Serialize(w io.Writer) error {
	err := common.WriteVarString(w, msg.GenesisAddress)
	if err != nil {
		return err
	}
	err = common.WriteUint32(w, msg.Height)
	if err != nil {
		return err
	}
	err = common.WriteVarUint(w, uint64(len(msg.OutPoints)))
	if err != nil {
		return err
	}
	for _, outPoint := range msg.OutPoints {
		err = outPoint.Serialize(w)
		if err != nil {
			return err
		}
	}
	err = common.WriteVarString(w, msg.Nonce)
	if err != nil {
		return err
	}
	return nil
}

func (msg *SendLastArbiterUsedUTXOMessage) Deserialize(r io.Reader) error {
	genesisAddress, err := common.ReadVarString(r)
	if err != nil {
		return err
	}
	msg.GenesisAddress = genesisAddress
	height, err := common.ReadUint32(r)
	if err != nil {
		return err
	}
	msg.Height = height

	lenOutPoints, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	var outPoints []core.OutPoint
	for i := uint64(0); i < lenOutPoints; i++ {
		var outPoint core.OutPoint
		err = outPoint.Deserialize(r)
		if err != nil {
			return err
		}
		outPoints = append(outPoints, outPoint)
	}
	msg.OutPoints = outPoints
	nonce, err := common.ReadVarString(r)
	if err != nil {
		return err
	}
	msg.Nonce = nonce
	return nil
}
