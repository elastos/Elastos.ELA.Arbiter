package cs

import (
	"bytes"
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/common"
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
		log.Warn("send withdraw transaction failed, move to finished db, txHash:", d.Tx.Hash().String())

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
		sidechain, ok := currentArbitrator.GetSideChainManager().GetChain(withdrawPayload.GenesisBlockAddress)
		if !ok {
			return errors.New("get side chain from withdraw payload failed")
		}
		sidechain.AddLastUsedOutPoints(newUsedUtxos)

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
	codeHashes, err := account.GetSigners(d.Tx.Programs[0].Code)
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
	payloadWithdraw, ok := d.Tx.Payload.(*payload.WithdrawFromSideChain)
	if !ok {
		return errors.New("unknown payload type")
	}

	clientFunc, ok := client.(DistributedNodeClientFunc)
	if !ok {
		return errors.New("unknown client function")
	}

	err := checkWithdrawTransaction(d.Tx, clientFunc)
	if err != nil {
		return err
	}

	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	sc, ok := currentArbitrator.GetSideChainManager().GetChain(payloadWithdraw.GenesisBlockAddress)
	if !ok {
		return errors.New("get side chain from GenesisBlockAddress failed")
	}

	if payloadWithdraw.BlockHeight > sc.GetLastUsedUtxoHeight() {
		var outPoints []types.OutPoint
		for _, input := range d.Tx.Inputs {
			outPoints = append(outPoints, input.Previous)
		}
		sc.AddLastUsedOutPoints(outPoints)
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

func checkWithdrawTransaction(txn *types.Transaction, clientFunc DistributedNodeClientFunc) error {
	payloadWithdraw, ok := txn.Payload.(*payload.WithdrawFromSideChain)
	if !ok {
		return errors.New("check withdraw transaction failed, unknown payload type")
	}

	sideChain, exchangeRate, err := clientFunc.GetSideChainAndExchangeRate(payloadWithdraw.GenesisBlockAddress)
	if err != nil {
		return err
	}

	//check genesis address
	var transactionHashes []string
	for _, hash := range payloadWithdraw.SideChainTransactionHashes {
		transactionHashes = append(transactionHashes, hash.String())
	}

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

			txid, err := common.Uint256FromHexString(tx.TxID)
			if err != nil {
				return errors.New("[checkWithdrawTransaction] failed, invalid txid")
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
				Txid: txid,
				WithdrawInfo: &base.WithdrawInfo{
					WithdrawAssets: withdrawAssets,
				},
			})
		}
	} else {
		txs = sideChainTxs
	}

	utxos, err := store.DbCache.UTXOStore.GetAddressUTXOsFromGenesisBlockAddress(payloadWithdraw.GenesisBlockAddress)
	if err != nil {
		return errors.New("get spender's UTXOs failed")
	}

	//check inputs
	var inputTotalAmount common.Fixed64
	for _, input := range txn.Inputs {
		isContained := false
		for _, utxo := range utxos {
			if utxo.Input.IsEqual(*input) {
				isContained = true
				inputTotalAmount += *utxo.Amount
				break
			}
		}
		if !isContained {
			return errors.New("check withdraw transaction failed, utxo is not from genesis address account")
		}
	}

	//check outputs and fee
	var outputTotalAmount common.Fixed64
	for _, output := range txn.Outputs {
		outputTotalAmount += output.Value
	}

	var totalFee common.Fixed64
	var oriOutputAmount common.Fixed64
	var totalCrossChainCount int
	for _, tx := range txs {
		for _, w := range tx.WithdrawInfo.WithdrawAssets {

			if *w.CrossChainAmount < 0 || *w.Amount <= 0 || *w.CrossChainAmount >= *w.Amount {
				return errors.New("check withdraw transaction failed, cross chain amount less than 0")
			}
			oriOutputAmount += common.Fixed64(float64(*w.CrossChainAmount) / exchangeRate)
			totalFee += common.Fixed64(float64(*w.Amount-*w.CrossChainAmount) / exchangeRate)
		}
		totalCrossChainCount += len(tx.WithdrawInfo.WithdrawAssets)
	}

	if inputTotalAmount != outputTotalAmount+totalFee {
		log.Info("inputTotalAmount-", inputTotalAmount, " outputTotalAmount-", outputTotalAmount, " totalFee-", totalFee)
		return errors.New("check withdraw transaction failed, input amount not equal output amount")
	}

	//check exchange rate
	genesisBlockProgramHash, err := common.Uint168FromAddress(payloadWithdraw.GenesisBlockAddress)
	if err != nil {
		return errors.New("check withdraw transaction failed, genesis block address to program hash failed")
	}
	var withdrawOutputAmount common.Fixed64
	var totalWithdrawOutputCount int
	for _, output := range txn.Outputs {
		if output.ProgramHash != *genesisBlockProgramHash {
			withdrawOutputAmount += output.Value
			totalWithdrawOutputCount++
		}
	}

	if totalCrossChainCount != totalWithdrawOutputCount {
		return errors.New("check withdraw transaction failed, cross chain count not equal withdraw output count")
	}

	if oriOutputAmount != withdrawOutputAmount {
		log.Info("oriOutputAmount-", oriOutputAmount, " withdrawOutputAmount-", withdrawOutputAmount)
		return errors.New("check withdraw transaction failed, exchange rate verify failed")
	}

	return nil
}
