package cs

import (
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type SchnorrWithdrawProposalContent struct {
	Tx *types.Transaction
}

func (c *SchnorrWithdrawProposalContent) SerializeUnsigned(w io.Writer) error {
	return c.Tx.SerializeUnsigned(w)
}

func (c *SchnorrWithdrawProposalContent) Serialize(w io.Writer) error {
	return c.Tx.Serialize(w)
}

func (c *SchnorrWithdrawProposalContent) Deserialize(r io.Reader) error {
	return c.Tx.Deserialize(r)
}

func (d *SchnorrWithdrawProposalContent) Hash() common.Uint256 {
	return d.Tx.Hash()
}

func (d *SchnorrWithdrawProposalContent) Check(client interface{}) error {
	clientFunc, ok := client.(DistributedNodeClientFunc)
	if !ok {
		return errors.New("unknown client function")
	}
	mainFunc := &arbitrator.MainChainFuncImpl{}
	height := store.DbCache.MainChainStore.CurrentHeight(store.QueryHeightCode)

	return checkSchnorrWithdrawTransaction(d.Tx, clientFunc, mainFunc, height)
}

func checkSchnorrWithdrawTransaction(
	txn *types.Transaction, clientFunc DistributedNodeClientFunc,
	mainFunc *arbitrator.MainChainFuncImpl, height uint32) error {
	if height < config.Parameters.SchnorrStartHeight {
		return errors.New("invalid schnorr withdraw transaction before start height")
	}

	switch txn.Payload.(type) {
	case *payload.WithdrawFromSideChain:
		err := checkWithdrawFromSideChainPayloadV1(txn, clientFunc, mainFunc)
		if err != nil {
			return err
		}
	case *payload.ReturnSideChainDepositCoin:
		err := checkReturnDepositTxPayload(txn, clientFunc)
		if err != nil {
			return err
		}
	default:
		return errors.New("check withdraw transaction failed, unknown payload type")
	}

	return nil
}
