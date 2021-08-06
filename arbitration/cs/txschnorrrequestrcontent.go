package cs

import (
	"errors"
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type SchnorrWithdrawRequestRProposalContent struct {
	Tx         *types.Transaction
	Publickeys [][]byte
}

func (c *SchnorrWithdrawRequestRProposalContent) SerializeUnsigned(w io.Writer) error {
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

	return nil
}

func (c *SchnorrWithdrawRequestRProposalContent) Serialize(w io.Writer) error {
	return c.SerializeUnsigned(w)
}

func (c *SchnorrWithdrawRequestRProposalContent) Deserialize(r io.Reader) error {
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

	return nil
}

func (d *SchnorrWithdrawRequestRProposalContent) Hash() common.Uint256 {
	return d.Tx.Hash()
}

func (d *SchnorrWithdrawRequestRProposalContent) Check(client interface{}) error {
	clientFunc, ok := client.(DistributedNodeClientFunc)
	if !ok {
		return errors.New("unknown client function")
	}
	mainFunc := &arbitrator.MainChainFuncImpl{}
	height := store.DbCache.MainChainStore.CurrentHeight(store.QueryHeightCode)

	return checkSchnorrWithdrawRequestRTransaction(d.Tx, clientFunc, mainFunc, height)
}

func checkSchnorrWithdrawRequestRTransaction(
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

func checkSchnorrWithdrawPayload(txn *types.Transaction,
	clientFunc DistributedNodeClientFunc, mainFunc *arbitrator.MainChainFuncImpl) error {
	if txn.PayloadVersion != payload.WithdrawFromSideChainVersionV2 {
		return errors.New("invalid schnorr withdraw payload version, not WithdrawFromSideChainVersionV2")
	}

	p, ok := txn.Payload.(*payload.WithdrawFromSideChain)
	if !ok {
		return errors.New("invalid transaction payload")
	}

	count := getTransactionAgreementArbitratorsCount(
		len(arbitrator.ArbitratorGroupSingleton.GetAllArbitrators()))

	if len(p.Signers) != count {
		return errors.New(fmt.Sprintf("invalid signer count, need:%d, current:%d", count, len(p.Signers)))
	}

	return checkWithdrawFromSideChainPayload(txn, clientFunc, mainFunc)
}

func checkSchnorrReturnDepositTxPayload(txn *types.Transaction,
	clientFunc DistributedNodeClientFunc) error {
	if txn.PayloadVersion != payload.ReturnSideChainDepositCoinVersionV1 {
		return errors.New("invalid schnorr return deposit payload version, not ReturnSideChainDepositCoinVersionV1")
	}

	p, ok := txn.Payload.(*payload.ReturnSideChainDepositCoin)
	if !ok {
		return errors.New("invalid transaction payload")
	}

	count := getTransactionAgreementArbitratorsCount(
		len(arbitrator.ArbitratorGroupSingleton.GetAllArbitrators()))

	if len(p.Signers) != count {
		return errors.New(fmt.Sprintf("invalid signer count, need:%d, current:%d", count, len(p.Signers)))
	}

	return checkReturnDepositTxPayload(txn, clientFunc)
}
