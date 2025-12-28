package cs

import (
	"bytes"
	"errors"
	"fmt"

	"io"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	elacommon "github.com/elastos/Elastos.ELA/core/types/common"
	it "github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type TxDistributedContent struct {
	Tx it.Transaction
}

func (d *TxDistributedContent) InitSign(newSign []byte) error {
	d.Tx.Programs()[0].Parameter = newSign
	return nil
}

func (d *TxDistributedContent) Submit() error {
	switch d.Tx.Payload().(type) {
	case *payload.WithdrawFromSideChain:
		return d.SubmitWithdrawTransaction()
	case *payload.ReturnSideChainDepositCoin:
		return d.SubmitReturnSideChainDepositCoin()
	case *payload.NFTDestroyFromSideChain:
		return d.SubmitNFTDestroyTransaction()
	default:
		return errors.New("received proposal feed back but transaction has invalid payload")
	}
}

func (d *TxDistributedContent) SubmitWithdrawTransaction() error {
	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	resp, err := currentArbitrator.SendWithdrawTransaction(d.Tx)

	pl, ok := d.Tx.Payload().(*payload.WithdrawFromSideChain)
	if !ok {
		return errors.New("invalid payload")
	}

	log.Info("Submit WithdrawFromSideChain transaction")
	var transactionHashes []string
	for _, hash := range pl.SideChainTransactionHashes {
		transactionHashes = append(transactionHashes, hash.String())
	}
	var dbStore store.DataStoreSideChain
	if d.Tx.PayloadVersion() == payload.WithdrawFromSideChainVersionV1 || d.Tx.PayloadVersion() == payload.WithdrawFromSideChainVersionV2 {
		var sideChain arbitrator.SideChain
		for _, output := range d.Tx.Outputs() {
			if output.Type != elacommon.OTWithdrawFromSideChain {
				continue
			}
			oPayload, ok := output.Payload.(*outputpayload.Withdraw)
			if !ok {
				return errors.New("invalid withdraw transaction output payload")
			}
			if sideChain == nil {
				var ok bool
				sideChain, ok = arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(oPayload.GenesisBlockAddress)
				if !ok || sideChain == nil {
					return errors.New("SubmitWithdrawTransaction Get side chain from genesis address failed.")
				}
			} else {
				if sideChain.GetKey() != oPayload.GenesisBlockAddress {
					return errors.New("invalid withdraw transaction GenesisBlockAddress")
				}
			}
		}
		dbStore = store.DbCache.GetDataStoreGenesisBlocAddress(sideChain.GetKey())
		if dbStore == nil {
			return errors.New("can't find db by genesis block hash ")
		}
	} else {
		dbStore = store.DbCache.GetDataStoreGenesisBlocAddress(pl.GenesisBlockAddress)
	}
	if dbStore == nil {
		return errors.New("can't find db by genesis block hash ")
	}

	if err != nil || resp.Error != nil && resp.Code != MCErrDoubleSpend {
		log.Warn("send withdraw transaction failed, move to finished db, txHash:", d.Tx.Hash().String(), ", code: ", resp.Code, ", result:", resp.Result)

		buf := new(bytes.Buffer)
		err := d.Tx.Serialize(buf)
		if err != nil {
			return errors.New("send withdraw transaction faild, invalid transaction")
		}

		err = dbStore.RemoveSideChainTxs(transactionHashes)
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
		var newUsedUtxos []elacommon.OutPoint
		for _, input := range d.Tx.Inputs() {
			newUsedUtxos = append(newUsedUtxos, input.Previous)
		}

		err = dbStore.RemoveSideChainTxs(transactionHashes)
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

func (d *TxDistributedContent) SubmitNFTDestroyTransaction() error {
	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	resp, err := currentArbitrator.SendWithdrawTransaction(d.Tx)

	pl, ok := d.Tx.Payload().(*payload.NFTDestroyFromSideChain)
	if !ok {
		return errors.New("invalid payload")
	}

	var ids []string
	for _, id := range pl.IDs {
		ids = append(ids, id.String())
	}
	chains := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetAllChains()
	genesisBlockAddress := ""
	for _, chain := range chains {
		chainConfig := chain.GetCurrentConfig()
		if chainConfig.GenesisBlock == pl.GenesisBlockHash.String() {
			genesisBlockAddress = chainConfig.GenesisBlockAddress
		}
	}
	dbStore := store.DbCache.GetDataStoreGenesisBlocAddress(genesisBlockAddress)
	if dbStore == nil {
		return errors.New("can't find db by genesis block hash ")
	}
	if err != nil || resp.Error != nil && resp.Code != MCErrDoubleSpend {
		log.Warn("send NFTDestroy transaction failed,  txHash:", d.Tx.Hash().String(), ", code: ", resp.Code, ", result:", resp.Result)
		err = dbStore.RemoveNFTDestroyTxs(ids)
		if err != nil {
			return errors.New("remove failed NFTDestroy transaction from db failed")
		}
		log.Warn("RemoveNFTDestroyTxs succed  ids ", ids)

	} else if resp.Error == nil && resp.Result != nil || resp.Error != nil && resp.Code == MCErrSidechainTxDuplicate {
		if resp.Error != nil {
			log.Info("send NFTDestroy transaction found has been processed, RemoveNFTDestroyTxs :", d.Tx.Hash().String())
		} else {
			log.Info("send NFTDestroy transaction succeed, RemoveNFTDestroyTxs, txHash:", d.Tx.Hash().String())
		}
		err = dbStore.RemoveNFTDestroyTxs(ids)
		if err != nil {
			return errors.New("remove succeed withdraw transaction from db failed")
		}
		log.Warn("RemoveNFTDestroyTxs succed  ids ", ids)

	} else {
		log.Warn("send NFTDestroy transaction failed, need to resend")
	}

	return nil
}

func (d *TxDistributedContent) SubmitReturnSideChainDepositCoin() error {
	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	resp, err := currentArbitrator.SendWithdrawTransaction(d.Tx)

	_, ok := d.Tx.Payload().(*payload.ReturnSideChainDepositCoin)
	if !ok {
		return errors.New("invalid payload")
	}

	log.Info("Submit return side chain deposit coin transaction")
	var transactionHashes []string
	var genesisAddresses []string
	for _, o := range d.Tx.Outputs() {
		if o.Type != elacommon.OTReturnSideChainDepositCoin {
			continue
		}
		opl, ok := o.Payload.(*outputpayload.ReturnSideChainDeposit)
		if !ok {
			return errors.New("invalid payload")
		}
		transactionHashes = append(transactionHashes, opl.DepositTransactionHash.String())
		genesisAddresses = append(genesisAddresses, opl.GenesisBlockAddress)
	}

	if err != nil {
		log.Warn("send return side chain deposit coin transaction err:", err)
	}
	if resp.Error != nil {
		log.Warn("send return side chain deposit coin transaction err:", resp.Error)
	}
	if err != nil || resp.Error != nil && resp.Code != MCErrDoubleSpend {
		log.Warn("failed to send return side chain deposit coin transaction, move to finished db, txHash:", d.Tx.Hash().String(), ", code: ", resp.Code, ", result:", resp.Result)

		buf := new(bytes.Buffer)
		err := d.Tx.Serialize(buf)
		if err != nil {
			return errors.New("failed to send return side chain deposit coin transaction , invalid transaction")
		}

		err = store.DbCache.MainChainStore.RemoveMainChainTxs(transactionHashes, genesisAddresses)
		if err != nil {
			return errors.New("failed to remove failed send return side chain deposit coin transaction from db")
		}
		// todo add to failed db
	} else if resp.Error == nil && resp.Result != nil || resp.Error != nil && resp.Code == MCErrSidechainTxDuplicate {
		if resp.Error != nil {
			log.Info("send send return side chain deposit coin transaction "+
				"found has been processed, move to finished db, txHash:", d.Tx.Hash().String())
		} else {
			log.Info("send send return side chain deposit coin transaction "+
				"succeed, move to finished db, txHash:", d.Tx.Hash().String())
		}

		err = store.DbCache.MainChainStore.RemoveMainChainTxs(transactionHashes, genesisAddresses)
		if err != nil {
			return errors.New("failed to remove succeed send return side chain deposit coin transaction from db")
		}
		// todo add to succeed db
	} else {
		log.Warn("failed to  send return side chain deposit coin transaction, need to resend")
	}

	return nil
}

func (d *TxDistributedContent) MergeSign(newSign []byte, targetCodeHash *common.Uint160) (int, error) {
	var signerIndex = -1
	codeHashes, err := account.GetCorssChainSigners(d.Tx.Programs()[0].Code)
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
	height := store.DbCache.MainChainStore.CurrentHeight(store.QueryHeightCode)

	return checkWithdrawTransaction(d.Tx, clientFunc, mainFunc, height)
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

func checkWithdrawFromSidechainPayload(txn it.Transaction,
	clientFunc DistributedNodeClientFunc, mainFunc *arbitrator.MainChainFuncImpl, payloadWithdraw *payload.WithdrawFromSideChain) error {
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

	frozenAddressMap := make(map[string]bool)
	for _, frozenAddress := range config.Parameters.FrozenAddresses {
		frozenAddressMap[frozenAddress] = true
	}
	dbStore := store.DbCache.GetDataStoreGenesisBlocAddress(payloadWithdraw.GenesisBlockAddress)
	if dbStore == nil {
		return errors.New(fmt.Sprintf("can't find db store by genesis block address:%s", payloadWithdraw.GenesisBlockAddress))
	}
	sideChainTxs, err := dbStore.GetSideChainTxsFromHashes(transactionHashes)
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
				if _, ok := frozenAddressMap[cs.CrossChainAddress]; ok {
					return errors.New("[checkWithdrawTransaction] invalid cross chain address in tx: frozen address, " + cs.CrossChainAddress)
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

	inputTotalAmount, err := mainFunc.GetAmountByInputs(txn.Inputs())
	if err != nil {
		return errors.New("get spender's UTXOs failed")
	}

	// check outputs and fee.
	var outputTotalAmount common.Fixed64
	withdrawOutputsMap := make(map[string]common.Fixed64)
	for _, output := range txn.Outputs() {
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
	for _, output := range txn.Outputs() {
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

func checkWithdrawTransaction(
	txn it.Transaction, clientFunc DistributedNodeClientFunc,
	mainFunc *arbitrator.MainChainFuncImpl, height uint32) error {
	switch pl := txn.Payload().(type) {
	case *payload.WithdrawFromSideChain:
		if height >= config.Parameters.NewCrossChainTransactionHeight {
			err := checkWithdrawFromSideChainPayloadV1(txn, clientFunc, mainFunc)
			if err != nil {
				return err
			}
		} else {
			err := checkWithdrawFromSidechainPayload(txn, clientFunc, mainFunc, pl)
			if err != nil {
				return err
			}
		}
	case *payload.ReturnSideChainDepositCoin:
		err := checkReturnDepositTxPayloadV0(txn, clientFunc)
		if err != nil {
			return err
		}
	case *payload.NFTDestroyFromSideChain:
		if height < config.Parameters.NFTStartHeight {
			return errors.New("NFT function not opened")
		}
		err := checkNFTDestroyFromSideChainTxPayload(txn, clientFunc, pl)
		if err != nil {
			return err
		}
	default:
		return errors.New("check withdraw transaction failed, unknown payload type")
	}

	return nil
}

func checkReturnDepositTxPayloadV0(txn it.Transaction, clientFunc DistributedNodeClientFunc) error {
	if txn.PayloadVersion() != payload.ReturnSideChainDepositCoinVersion {
		return errors.New("invalid schnorr return deposit payload version, not ReturnSideChainDepositCoinVersion")
	}

	return checkReturnDepositTxPayload(txn, clientFunc)
}

//NFTDestroyFromSideChain
func checkNFTDestroyFromSideChainTxPayload(txn it.Transaction, clientFunc DistributedNodeClientFunc,
	nftDestroyPayload *payload.NFTDestroyFromSideChain) error {
	if txn.PayloadVersion() != payload.NFTDestroyFromSideChainVersion {
		return errors.New("invalid NFTDestroyFromSideChain Tx payload version, not NFTDestroyFromSideChainVersion")
	}

	return checkNFTDestroyFromSideChainPayload(txn, clientFunc, nftDestroyPayload)
}

func checkReturnDepositTxPayload(txn it.Transaction, clientFunc DistributedNodeClientFunc) error {
	// check if withdraw transactions exist in db, if not found then will check
	// by the rpc interface of the side chain.
	log.Info("[checkReturnDepositTxPayload], need to get side chain transaction from rpc")
	var outputTotalAmount common.Fixed64
	for _, o := range txn.Outputs() {
		outputTotalAmount += o.Value

		if o.Type != elacommon.OTReturnSideChainDepositCoin {
			continue
		}
		opl, ok := o.Payload.(*outputpayload.ReturnSideChainDeposit)
		if !ok {
			return errors.New("[checkReturnDepositTxPayload], invalid output payload")
		}
		sideChain, _, err := clientFunc.GetSideChainAndExchangeRate(opl.GenesisBlockAddress)
		if err != nil {
			return err
		}

		exist, err := sideChain.GetFailedDepositTransaction(opl.DepositTransactionHash.String())
		if err != nil || !exist {
			return errors.New("[checkReturnDepositTxPayload] failed, unknown side chain transactions")
		}
		txnBytes, err := common.HexStringToBytes(opl.DepositTransactionHash.String())
		if err != nil {
			return errors.New("[checkReturnDepositTxPayload] tx hash can not reversed")
		}
		reversedTxnBytes := common.BytesReverse(txnBytes)
		reversedTx := common.BytesToHexString(reversedTxnBytes)
		originTx, err := rpc.GetTransaction(reversedTx, config.Parameters.MainNode.Rpc)
		if err != nil {
			return errors.New("[checkReturnDepositTxPayload] failed to get origin tx from main chain:" + err.Error())
		}
		referTxid := originTx.Inputs()[0].Previous.TxID
		referIndex := originTx.Inputs()[0].Previous.Index
		referReversedTx := common.BytesToHexString(common.BytesReverse(referTxid.Bytes()))
		referTxn, err := rpc.GetTransaction(referReversedTx, config.Parameters.MainNode.Rpc)
		if err != nil {
			log.Errorf("[checkReturnDepositTxPayload] referReversedTx", err.Error())
			break
		}
		_, ok = originTx.Payload().(*payload.TransferCrossChainAsset)
		if !ok {
			return errors.New("[checkReturnDepositTxPayload] invalid payload type need TransferCrossChainAsset")
		}
		address, err := referTxn.Outputs()[referIndex].ProgramHash.ToAddress()
		if err != nil {
			return errors.New("[checkReturnDepositTxPayload] ProgramHash can not transfer to address")
		}
		crossChainHash, err := common.Uint168FromAddress(opl.GenesisBlockAddress)
		if err != nil {
			return err
		}

		var depositAmount common.Fixed64
		for _, output := range originTx.Outputs() {
			if bytes.Compare(output.ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) != 0 {
				continue
			}
			if !crossChainHash.IsEqual(output.ProgramHash) {
				continue
			}

			depositAmount += output.Value
		}
		if o.Value != depositAmount-config.Parameters.ReturnDepositTransactionFee {
			return errors.New("[checkReturnDepositTxPayload] invalid output amount")
		}
		outputAddr, err := o.ProgramHash.ToAddress()
		if err != nil {
			return errors.New("[checkReturnDepositTxPayload] invalid output address")
		}
		if outputAddr != address {
			return errors.New("[checkReturnDepositTxPayload] address is:" +
				outputAddr + "should be:" + address)
		}

	}

	return nil
}

func checkWithdrawFromSideChainPayloadV1(txn it.Transaction,
	clientFunc DistributedNodeClientFunc, mainFunc *arbitrator.MainChainFuncImpl) error {
	if txn.PayloadVersion() != payload.WithdrawFromSideChainVersionV1 {
		return errors.New("invalid withdraw payload version, not WithdrawFromSideChainVersionV1")
	}

	return checkWithdrawFromSideChainPayload(txn, clientFunc, mainFunc)
}

func checkWithdrawFromSideChainPayload(txn it.Transaction,
	clientFunc DistributedNodeClientFunc, mainFunc *arbitrator.MainChainFuncImpl) error {
	var transactionHashes []string
	var sideChain arbitrator.SideChain
	var exchangeRate float64
	for i, output := range txn.Outputs() {
		log.Info("checkWithdrawFromSideChainPayloadV1 output[", i, "]", output.String())
		if output.Type != elacommon.OTWithdrawFromSideChain {
			continue
		}
		oPayload, ok := output.Payload.(*outputpayload.Withdraw)
		if !ok {
			return errors.New("invalid withdraw transaction output payload")
		}
		transactionHashes = append(transactionHashes, oPayload.SideChainTransactionHash.String())
		if sideChain == nil {
			var err error
			sideChain, exchangeRate, err = clientFunc.GetSideChainAndExchangeRate(oPayload.GenesisBlockAddress)
			if err != nil {
				return err
			}
		} else {
			if sideChain.GetKey() != oPayload.GenesisBlockAddress {
				return errors.New("invalid withdraw transaction GenesisBlockAddress")
			}
		}
	}

	if len(transactionHashes) == 0 {
		return errors.New("invalid withdraw transaction count")
	}

	genesisAddress := sideChain.GetKey()
	// check if withdraw transactions exist in db, if not found then will check
	// by the rpc interface of the side chain.

	var txs []*base.WithdrawTx
	dbStore := store.DbCache.GetDataStoreGenesisBlocAddress(genesisAddress)
	if dbStore == nil {
		return errors.New(fmt.Sprintf("can't find db store by genesis block address:%s", genesisAddress))
	}
	sideChainTxs, err := dbStore.GetSideChainTxsFromHashes(transactionHashes)
	if err != nil || len(sideChainTxs) != len(transactionHashes) {
		log.Info("[checkWithdrawTransaction], need to get side chain transaction from rpc")
		for _, txHash := range transactionHashes {
			tx, err := sideChain.GetWithdrawTransaction(txHash)
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

	inputTotalAmount, err := mainFunc.GetAmountByInputs(txn.Inputs())
	if err != nil {
		return errors.New("get spender's UTXOs failed")
	}

	// check outputs and fee.
	var outputTotalAmount common.Fixed64
	withdrawOutputsMap := make(map[string]common.Fixed64)
	for _, output := range txn.Outputs() {
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
			
	frozenAddressMap := make(map[string]bool)
	for _, frozenAddress := range config.Parameters.FrozenAddresses {
		frozenAddressMap[frozenAddress] = true
	}
	for _, tx := range txs {
		for _, w := range tx.WithdrawInfo.WithdrawAssets {
			if _, ok := frozenAddressMap[w.TargetAddress]; ok {
				return errors.New("[checkWithdrawTransaction] invalid cross chain address in tx: frozen address, " + w.TargetAddress)
			}
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
	genesisBlockProgramHash, err := common.Uint168FromAddress(genesisAddress)
	if err != nil {
		return errors.New("check withdraw transaction failed, genesis " +
			"block address to program hash failed")
	}
	var withdrawOutputAmount common.Fixed64
	var totalWithdrawAmount int
	for _, output := range txn.Outputs() {
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

func checkNFTDestroyFromSideChainPayload(txn it.Transaction, clientFunc DistributedNodeClientFunc,
	nftDestroyPayload *payload.NFTDestroyFromSideChain) error {

	chains := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetAllChains()
	genesisBlockAddress := ""
	for _, chain := range chains {
		chainConfig := chain.GetCurrentConfig()
		if chainConfig.GenesisBlock == nftDestroyPayload.GenesisBlockHash.String() {
			genesisBlockAddress = chainConfig.GenesisBlockAddress
		}
	}
	_, _, err := clientFunc.GetSideChainAndExchangeRate(genesisBlockAddress)
	if err != nil {
		return err
	}
	return nil
}
