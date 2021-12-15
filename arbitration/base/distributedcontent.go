package base

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

type DistributedContent interface {
	Check(clientFunc interface{}) error
	Submit() error
	InitSign(newSign []byte) error
	MergeSign(newSign []byte, targetCodeHash *common.Uint160) (int, error)

	Serialize(w io.Writer) error
	SerializeUnsigned(w io.Writer) error
	Deserialize(r io.Reader) error
	DeserializeUnsigned(r io.Reader) error

	Hash() common.Uint256
}
