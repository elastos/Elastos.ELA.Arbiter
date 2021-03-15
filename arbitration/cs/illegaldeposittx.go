package cs

import (
	"bytes"
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type IllegalDepositTx struct {
	DepositTxs *payload.IllegalDepositTxs

	hash *common.Uint256
}

func (i *IllegalDepositTx) Check(clientFunc interface{}) error {
	sideChain, ok := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(i.DepositTxs.GenesisBlockAddress)
	if !ok || sideChain == nil {
		return errors.New("get side chain from genesis address failed when check illegal deposit evidence")
	}
	sideChain.CheckIllegalDepositTx(i.DepositTxs.DepositTxs)
	return nil
}

func (i *IllegalDepositTx) CurrentBlockHeight() (uint32, error) {
	return i.DepositTxs.Height, nil
}

func (i *IllegalDepositTx) Deserialize(r io.Reader) error {
	return i.DepositTxs.Deserialize(r, payload.SidechainIllegalDataVersion)
}

func (i *IllegalDepositTx) DeserializeUnsigned(r io.Reader) error {
	return i.DepositTxs.DeserializeUnsigned(r, payload.SidechainIllegalDataVersion)
}

func (i *IllegalDepositTx) Hash() common.Uint256 {
	if i.hash == nil {
		buf := new(bytes.Buffer)
		i.SerializeUnsigned(buf)
		hash := common.Uint256(common.Sha256D(buf.Bytes()))
		i.hash = &hash
	}
	return *i.hash
}

func (i *IllegalDepositTx) InitSign(newSign []byte) error {
	i.DepositTxs.Signs = [][]byte{newSign}
	return nil
}

func (i *IllegalDepositTx) MergeSign(newSign []byte, targetCodeHash *common.Uint160) (int, error) {
	i.DepositTxs.Signs = append(i.DepositTxs.Signs, newSign)
	return len(i.DepositTxs.Signs), nil
}

func (i *IllegalDepositTx) Serialize(w io.Writer) error {
	return i.DepositTxs.Serialize(w, payload.IllegalDepositTxsVersion)
}

func (i *IllegalDepositTx) SerializeUnsigned(w io.Writer) error {
	return i.DepositTxs.SerializeUnsigned(w, payload.IllegalDepositTxsVersion)
}

func (i *IllegalDepositTx) Submit() error {
	var err error
	buf := new(bytes.Buffer)
	if err = i.DepositTxs.Serialize(buf, payload.IllegalDepositTxsVersion); err != nil {
		return err
	}

	content := common.BytesToHexString(buf.Bytes())
	if _, err = rpc.CallAndUnmarshalResponse("submitillegaldeposittxsdata",
		rpc.Param("illegaldeposittxsdata", content), config.Parameters.MainNode.Rpc); err != nil {
		return err
	}
	return nil
}
