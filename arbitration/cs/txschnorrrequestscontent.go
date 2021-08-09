package cs

import (
	"errors"
	"io"
	"math/big"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type SchnorrWithdrawRequestSProposalContent struct {
	Tx         *types.Transaction
	Publickeys [][]byte
	E          *big.Int
	S          *big.Int
}

func (c *SchnorrWithdrawRequestSProposalContent) SerializeUnsigned(w io.Writer, feedback bool) error {
	if err := c.Tx.SerializeUnsigned(w); err != nil {
		return errors.New("failed tto serialize transaction")
	}

	if err := common.WriteVarUint(w, uint64(len(c.Publickeys))); err != nil {
		return errors.New("failed to write count of public keys")
	}

	for _, pk := range c.Publickeys {
		if err := common.WriteVarBytes(w, pk); err != nil {
			return errors.New("failed to serialize public key")
		}
	}
	if err := common.WriteVarBytes(w, c.E.Bytes()); err != nil {
		return err
	}
	if feedback {
		if err := common.WriteVarBytes(w, c.S.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (c *SchnorrWithdrawRequestSProposalContent) Serialize(w io.Writer, feedback bool) error {
	return c.SerializeUnsigned(w, feedback)
}

func (c *SchnorrWithdrawRequestSProposalContent) Deserialize(r io.Reader, feedback bool) error {
	if err := c.Tx.Deserialize(r); err != nil {
		return errors.New("failed to deserialize transaction")
	}

	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	c.Publickeys = make([][]byte, 0)
	for i := uint64(0); i < count; i++ {
		pk, err := common.ReadVarBytes(r, 32, "pk")
		if err != nil {
			return err
		}
		c.Publickeys = append(c.Publickeys, pk)
	}
	e, err := common.ReadVarBytes(r, 64, "e")
	if err != nil {
		return err
	}
	c.E = new(big.Int).SetBytes(e)

	if feedback {
		s, err := common.ReadVarBytes(r, 64, "s")
		if err != nil {
			return err
		}
		c.S = new(big.Int).SetBytes(s)
	}

	return nil
}

func (d *SchnorrWithdrawRequestSProposalContent) Hash() common.Uint256 {
	return d.Tx.Hash()
}

func (d *SchnorrWithdrawRequestSProposalContent) Check(client interface{}) error {
	clientFunc, ok := client.(DistributedNodeClientFunc)
	if !ok {
		return errors.New("unknown client function")
	}
	mainFunc := &arbitrator.MainChainFuncImpl{}
	height := store.DbCache.MainChainStore.CurrentHeight(store.QueryHeightCode)

	return checkSchnorrWithdrawRequestSTransaction(d.Tx, clientFunc, mainFunc, height)
}

func checkSchnorrWithdrawRequestSTransaction(
	txn *types.Transaction, clientFunc DistributedNodeClientFunc,
	mainFunc *arbitrator.MainChainFuncImpl, height uint32) error {
	if height < config.Parameters.SchnorrStartHeight {
		return errors.New("invalid schnorr withdraw transaction before start height")
	}

	switch txn.Payload.(type) {
	case *payload.WithdrawFromSideChain:
		err := checkSchnorrWithdrawPayload(txn, clientFunc, mainFunc)
		if err != nil {
			return err
		}
	case *payload.ReturnSideChainDepositCoin:
		err := checkSchnorrReturnDepositTxPayload(txn, clientFunc)
		if err != nil {
			return err
		}
	default:
		return errors.New("check withdraw transaction failed, unknown payload type")
	}

	return nil
}
