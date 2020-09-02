package cs

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type TxDistributedContent struct {
	Tx *types.Transaction
}

func (d *TxDistributedContent) InitSign(newSign []byte) error {
	d.Tx.Programs[0].Parameter = newSign
	return nil
}

func (d *TxDistributedContent) Submit() error {
	withdrawPayload, ok := d.Tx.Payload.(*payload.WithdrawFromSideChain)
	if !ok {
		return errors.New("received proposal feed back but withdraw transaction has invalid payload")
	}

	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	resp, err := currentArbitrator.SendWithdrawTransaction(d.Tx)

	var transactionHashes []string
	for _, hash := range withdrawPayload.SideChainTransactionHashes {
		transactionHashes = append(transactionHashes, hash.String())
	}

	if err != nil || resp.Error != nil && resp.Code != MCErrDoubleSpend {
		log.Warn("send withdraw transaction failed, move to finished db, txHash:", d.Tx.Hash().String(), ", code: ", resp.Code, ", result:", resp.Result)

		buf := new(bytes.Buffer)
		err := d.Tx.Serialize(buf)
		if err != nil {
			return errors.New("send withdraw transaction faild, invalid transaction")
		}

		err = store.DbCache.SideChainStore.RemoveSideChainTxs(transactionHashes)
		if err != nil {
			return errors.New("remove failed withdraw transaction from db failed")
		}
		err = store.FinishedTxsDbCache.AddFailedWithdrawTxs(transactionHashes, buf.Bytes())
		if err != nil {
			return errors.New("add failed withdraw transaction into finished db failed")
		}
	} else if resp.Error == nil && resp.Result != nil || resp.Error != nil && resp.Code == MCErrSidechainTxDuplicate {
		if resp.Error != nil {
			log.Info("send withdraw transaction found has been processed, move to finished db, txHash:", d.Tx.Hash().String())
		} else {
			log.Info("send withdraw transaction succeed, move to finished db, txHash:", d.Tx.Hash().String())
		}
		var newUsedUtxos []types.OutPoint
		for _, input := range d.Tx.Inputs {
			newUsedUtxos = append(newUsedUtxos, input.Previous)
		}

		err = store.DbCache.SideChainStore.RemoveSideChainTxs(transactionHashes)
		if err != nil {
			return errors.New("remove succeed withdraw transaction from db failed")
		}
		err = store.FinishedTxsDbCache.AddSucceedWithdrawTxs(transactionHashes)
		if err != nil {
			return errors.New("add succeed withdraw transaction into finished db failed")
		}
	} else {
		log.Warn("send withdraw transaction failed, need to resend")
	}

	return nil
}

func (d *TxDistributedContent) MergeSign(newSign []byte, targetCodeHash *common.Uint160) (int, error) {
	var signerIndex = -1
	codeHashes, err := account.GetCorssChainSigners(d.Tx.Programs[0].Code)
	if err != nil {
		return 0, err
	}
	for i, programHash := range codeHashes {
		if targetCodeHash.IsEqual(*programHash) {
			signerIndex = i
			break
		}
	}
	if signerIndex == -1 {
		return 0, errors.New("invalid multi sign signer")
	}

	signedCount, err := base.MergeSignToTransaction(newSign, signerIndex, d.Tx)
	if err != nil {
		return 0, err
	}

	return signedCount, nil
}

func (d *TxDistributedContent) Check(client interface{}) error {
	clientFunc, ok := client.(DistributedNodeClientFunc)
	if !ok {
		return errors.New("unknown client function")
	}
	mainFunc := &arbitrator.MainChainFuncImpl{}
	err := checkWithdrawTransaction(d.Tx, clientFunc, mainFunc)
	if err != nil {
		return err
	}
	return nil
}

func (d *TxDistributedContent) CurrentBlockHeight() (uint32, error) {
	withdrawPayload, ok := d.Tx.Payload.(*payload.WithdrawFromSideChain)
	if !ok {
		return 0, errors.New("invalid payload type")
	}
	return withdrawPayload.BlockHeight, nil
}

func (d *TxDistributedContent) Serialize(w io.Writer) error {
	return d.Tx.Serialize(w)
}

func (d *TxDistributedContent) SerializeUnsigned(w io.Writer) error {
	return d.Tx.SerializeUnsigned(w)
}

func (d *TxDistributedContent) Deserialize(r io.Reader) error {
	return d.Tx.Deserialize(r)
}

func (d *TxDistributedContent) DeserializeUnsigned(r io.Reader) error {
	return d.Tx.DeserializeUnsigned(r)
}

func (d *TxDistributedContent) Hash() common.Uint256 {
	return d.Tx.Hash()
}

func checkWithdrawTransaction(txn *types.Transaction,
	clientFunc DistributedNodeClientFunc, mainFunc *arbitrator.MainChainFuncImpl) error {
	payloadWithdraw, ok := txn.Payload.(*payload.WithdrawFromSideChain)
	if !ok {
		return errors.New("check withdraw transaction failed, unknown payload type")
	}

	// check if side chain exist.
	sideChain, exchangeRate, err := clientFunc.GetSideChainAndExchangeRate(payloadWithdraw.GenesisBlockAddress)
	if err != nil {
		return err
	}

	var transactionHashes []string
	for _, hash := range payloadWithdraw.SideChainTransactionHashes {
		transactionHashes = append(transactionHashes, hash.String())
	}

	// check if withdraw transactions exist in db, if not found then will check
	// by the rpc interface of the side chain.
	var txs []*base.WithdrawTx
	sideChainTxs, err := store.DbCache.SideChainStore.GetSideChainTxsFromHashesAndGenesisAddress(
		transactionHashes, payloadWithdraw.GenesisBlockAddress)
	if err != nil || len(sideChainTxs) != len(payloadWithdraw.SideChainTransactionHashes) {
		log.Info("[checkWithdrawTransaction], need to get side chain transaction from rpc")
		for _, txHash := range payloadWithdraw.SideChainTransactionHashes {
			tx, err := sideChain.GetWithdrawTransaction(txHash.String())
			if err != nil {
				return errors.New("[checkWithdrawTransaction] failed, unknown side chain transactions")
			}

			txID, err := common.Uint256FromHexString(tx.TxID)
			if err != nil {
				return errors.New("[checkWithdrawTransaction] failed, invalid txID")
			}

			var withdrawAssets []*base.WithdrawAsset
			for _, cs := range tx.CrossChainAssets {
				csAmount, err := common.StringToFixed64(cs.CrossChainAmount)
				if err != nil {
					return errors.New("[checkWithdrawTransaction] invalid cross chain amount in tx")
				}
				opAmount, err := common.StringToFixed64(cs.OutputAmount)
				if err != nil {
					return errors.New("[checkWithdrawTransaction] invalid output amount in tx")
				}
				withdrawAssets = append(withdrawAssets, &base.WithdrawAsset{
					TargetAddress:    cs.CrossChainAddress,
					Amount:           opAmount,
					CrossChainAmount: csAmount,
				})
			}

			txs = append(txs, &base.WithdrawTx{
				Txid: txID,
				WithdrawInfo: &base.WithdrawInfo{
					WithdrawAssets: withdrawAssets,
				},
			})
		}
	} else {
		txs = sideChainTxs
	}

	inputTotalAmount, err := mainFunc.GetAmountByInputs(txn.Inputs)
	if err != nil {
		return errors.New("get spender's UTXOs failed")
	}

	// check outputs and fee.
	var outputTotalAmount common.Fixed64
	withdrawOutputsMap := make(map[string]common.Fixed64)
	for _, output := range txn.Outputs {
		outputTotalAmount += output.Value

		if contract.PrefixType(output.ProgramHash[0]) == contract.PrefixCrossChain {
			continue
		}
		addr, err := output.ProgramHash.ToAddress()
		if err != nil {
			continue
		}
		amount, ok := withdrawOutputsMap[addr]
		if ok {
			withdrawOutputsMap[addr] = amount + output.Value
		} else {
			withdrawOutputsMap[addr] = output.Value
		}
	}

	var totalFee common.Fixed64
	var oriOutputAmount common.Fixed64
	var totalCrossChainAmount int
	crossChainOutputsMap := make(map[string]common.Fixed64)
	for _, tx := range txs {
		for _, w := range tx.WithdrawInfo.WithdrawAssets {
			if *w.CrossChainAmount < 0 || *w.Amount <= 0 ||
				*w.Amount-*w.CrossChainAmount <= 0 ||
				*w.CrossChainAmount >= *w.Amount {
				return errors.New("check withdraw transaction " +
					"failed, cross chain amount less than 0")
			}
			oriOutputAmount += common.Fixed64(float64(*w.CrossChainAmount) / exchangeRate)
			totalFee += common.Fixed64(float64(*w.Amount-*w.CrossChainAmount) / exchangeRate)

			amount, ok := crossChainOutputsMap[w.TargetAddress]
			if ok {
				crossChainOutputsMap[w.TargetAddress] = amount + *w.CrossChainAmount
			} else {
				crossChainOutputsMap[w.TargetAddress] = *w.CrossChainAmount
			}
		}
		totalCrossChainAmount += len(tx.WithdrawInfo.WithdrawAssets)
	}

	if inputTotalAmount != outputTotalAmount+totalFee {
		log.Info("inputTotalAmount-", inputTotalAmount,
			" outputTotalAmount-", outputTotalAmount, " totalFee-", totalFee)
		return errors.New("check withdraw transaction failed, input " +
			"amount not equal output amount")
	}

	// check exchange rate.
	genesisBlockProgramHash, err := common.Uint168FromAddress(payloadWithdraw.GenesisBlockAddress)
	if err != nil {
		return errors.New("check withdraw transaction failed, genesis " +
			"block address to program hash failed")
	}
	var withdrawOutputAmount common.Fixed64
	var totalWithdrawAmount int
	for _, output := range txn.Outputs {
		if output.ProgramHash != *genesisBlockProgramHash {
			withdrawOutputAmount += output.Value
			totalWithdrawAmount++
		}
	}

	if totalCrossChainAmount != totalWithdrawAmount ||
		len(crossChainOutputsMap) != len(withdrawOutputsMap) {
		return errors.New("check withdraw transaction failed, cross chain " +
			"amount not equal withdraw total amount")
	}

	for k, v := range withdrawOutputsMap {
		amount, ok := crossChainOutputsMap[k]
		if !ok || common.Fixed64(float64(amount)/exchangeRate) != v {
			return fmt.Errorf("check withdraw transaction failed, addr"+
				" %s amount is invalid, real is %s, need to be %s", k,
				v.String(), amount.String())
		}
	}

	if oriOutputAmount != withdrawOutputAmount {
		log.Info("oriOutputAmount-", oriOutputAmount, " withdrawOutputAmount-", withdrawOutputAmount)
		return errors.New("check withdraw transaction failed, exchange rate verify failed")
	}

	return nil
}
