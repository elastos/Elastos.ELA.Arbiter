package cs

import (
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/common"
	elatx "github.com/elastos/Elastos.ELA/core/transaction"
	it "github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type SchnorrWithdrawRequestSProposalContent struct {
	NonceHash  common.Uint256
	Tx         it.Transaction
	Publickeys [][]byte
	E          *big.Int
	S          *big.Int
}

func (c *SchnorrWithdrawRequestSProposalContent) SerializeUnsigned(w io.Writer, feedback bool) error {
	if err := c.NonceHash.Serialize(w); err != nil {
		return err
	}

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
	if err := c.NonceHash.Deserialize(r); err != nil {
		return err
	}

	tx, err := elatx.GetTransactionByBytes(r)
	if err != nil {
		return err
	}
	if err := tx.DeserializeUnsigned(r); err != nil {
		return errors.New("failed to deserialize transaction")
	}
	c.Tx = tx

	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	c.Publickeys = make([][]byte, 0)
	for i := uint64(0); i < count; i++ {
		pk, err := common.ReadVarBytes(r, 33, "pk")
		if err != nil {
			return err
		}
		c.Publickeys = append(c.Publickeys, pk)
	}

	e, err := common.ReadVarBytes(r, 65, "e")
	if err != nil {
		return err
	}
	c.E = new(big.Int).SetBytes(e)

	if feedback {
		s, err := common.ReadVarBytes(r, 65, "s")
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
	txn it.Transaction, clientFunc DistributedNodeClientFunc,
	mainFunc *arbitrator.MainChainFuncImpl, height uint32) error {
	if height < config.Parameters.SchnorrStartHeight {
		return errors.New("invalid schnorr withdraw transaction before start height")
	}

	switch txn.Payload().(type) {
	case *payload.WithdrawFromSideChain:
		err := checkSchnorrWithdrawPayload(txn, clientFunc, mainFunc)
		if err != nil {
			log.Error("check schnorr withdraw payload err:", err)
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

func checkSchnorrWithdrawPayload(txn it.Transaction,
	clientFunc DistributedNodeClientFunc, mainFunc *arbitrator.MainChainFuncImpl) error {
	if txn.PayloadVersion() != payload.WithdrawFromSideChainVersionV2 {
		return errors.New("invalid schnorr withdraw payload version, not WithdrawFromSideChainVersionV2")
	}

	p, ok := txn.Payload().(*payload.WithdrawFromSideChain)
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

func checkSchnorrReturnDepositTxPayload(txn it.Transaction,
	clientFunc DistributedNodeClientFunc) error {
	if txn.PayloadVersion() != payload.ReturnSideChainDepositCoinVersionV1 {
		return errors.New("invalid schnorr return deposit payload version, not ReturnSideChainDepositCoinVersionV1")
	}

	p, ok := txn.Payload().(*payload.ReturnSideChainDepositCoin)
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
