package cs

import (
	"bytes"
	"errors"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Utility/common"
	ela "github.com/elastos/Elastos.ELA/core"
)

type DistributedNodeClient struct {
	P2pCommand string
}

type DistributedNodeClientFunc interface {
	GetSideChainAndExchangeRate(genesisAddress string) (SideChain, float64, error)
}

func (client *DistributedNodeClient) GetSideChainAndExchangeRate(genesisAddress string) (SideChain, float64, error) {
	sideChain, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(genesisAddress)
	if !ok || sideChain == nil {
		return nil, 0, errors.New("Get side chain from genesis address failed.")
	}
	rate, err := sideChain.GetExchangeRate()
	if err != nil {
		return nil, 0, err
	}
	return sideChain, rate, nil
}

func (client *DistributedNodeClient) SignProposal(item *DistributedItem) error {
	return item.Sign(ArbitratorGroupSingleton.GetCurrentArbitrator(), true, &DistrubutedItemFuncImpl{})
}

func (client *DistributedNodeClient) OnReceivedProposal(content []byte) error {
	log.Debug("[Client][OnReceivedProposal] start")
	defer log.Debug("[Client][OnReceivedProposal] end")

	transactionItem := &DistributedItem{}
	if err := transactionItem.Deserialize(bytes.NewReader(content)); err != nil {
		return err
	}

	if transactionItem.IsFeedback() {
		return nil
	}

	payloadWithdraw, ok := transactionItem.ItemContent.Payload.(*ela.PayloadWithdrawFromSideChain)
	if !ok {
		return errors.New("Unknown payload type.")
	}

	err := checkWithdrawTransaction(transactionItem.ItemContent, client)
	if err != nil {
		return err
	}

	currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()
	sc, ok := currentArbitrator.GetSideChainManager().GetChain(payloadWithdraw.GenesisBlockAddress)
	if !ok {
		return errors.New("Get side chain from GenesisBlockAddress failed")
	}

	if payloadWithdraw.BlockHeight > sc.GetLastUsedUtxoHeight() {
		var outPoints []ela.OutPoint
		for _, input := range transactionItem.ItemContent.Inputs {
			outPoints = append(outPoints, input.Previous)
		}
		sc.AddLastUsedOutPoints(outPoints)
	}

	if err := client.SignProposal(transactionItem); err != nil {
		return err
	}

	if err := client.Feedback(transactionItem); err != nil {
		return err
	}

	return nil
}

func (client *DistributedNodeClient) Feedback(item *DistributedItem) error {
	ar := ArbitratorGroupSingleton.GetCurrentArbitrator()
	item.TargetArbitratorPublicKey = ar.GetPublicKey()

	programHash, err := StandardAcccountPublicKeyToProgramHash(item.TargetArbitratorPublicKey)
	if err != nil {
		return err
	}
	item.TargetArbitratorProgramHash = programHash

	messageReader := new(bytes.Buffer)
	err = item.Serialize(messageReader)
	if err != nil {
		return errors.New("Send complaint failed.")
	}

	client.broadcast(messageReader.Bytes())
	return nil
}

func (client *DistributedNodeClient) broadcast(message []byte) {
	msg := &SignMessage{
		Command: client.P2pCommand,
		Content: message,
	}
	P2PClientSingleton.AddMessageHash(P2PClientSingleton.GetMessageHash(msg))
	P2PClientSingleton.Broadcast(msg)
}

func checkWithdrawTransaction(txn *ela.Transaction, clientFunc DistributedNodeClientFunc) error {
	payloadWithdraw, ok := txn.Payload.(*ela.PayloadWithdrawFromSideChain)
	if !ok {
		return errors.New("Check withdraw transaction failed, unknown payload type")
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

	var txs []*WithdrawTx
	sideChainTxs, err := store.DbCache.SideChainStore.GetSideChainTxsFromHashesAndGenesisAddress(
		transactionHashes, payloadWithdraw.GenesisBlockAddress)
	if err != nil || len(sideChainTxs) != len(payloadWithdraw.SideChainTransactionHashes) {
		log.Info("[checkWithdrawTransaction], need to get side chain transaction from rpc")
		for _, txHash := range payloadWithdraw.SideChainTransactionHashes {
			tx, err := sideChain.GetWithdrawTransaction(txHash.String())
			if err != nil {
				return errors.New("[checkWithdrawTransaction] failed, unknown side chain transachtions")
			}

			txid, err := common.Uint256FromHexString(tx.TxID)
			if err != nil {
				return errors.New("[checkWithdrawTransaction] failed, invalid txid")
			}

			var withdrawAssets []*WithdrawAsset
			for _, cs := range tx.CrossChainAssets {
				csAmount, err := common.StringToFixed64(cs.CrossChainAmount)
				if err != nil {
					return errors.New("[checkWithdrawTransaction] invlaid cross chain amount in tx")
				}
				opAmount, err := common.StringToFixed64(cs.OutputAmount)
				if err != nil {
					return errors.New("[checkWithdrawTransaction] invlaid output amount in tx")
				}
				withdrawAssets = append(withdrawAssets, &WithdrawAsset{
					TargetAddress:    cs.CrossChainAddress,
					Amount:           opAmount,
					CrossChainAmount: csAmount,
				})
			}

			txs = append(txs, &WithdrawTx{
				Txid: txid,
				WithdrawInfo: &WithdrawInfo{
					WithdrawAssets: withdrawAssets,
				},
			})
		}
	} else {
		txs = sideChainTxs
	}

	utxos, err := store.DbCache.UTXOStore.GetAddressUTXOsFromGenesisBlockAddress(payloadWithdraw.GenesisBlockAddress)
	if err != nil {
		return errors.New("Get spender's UTXOs failed")
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
			return errors.New("Check withdraw transaction failed, utxo is not from genesis address account")
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
				return errors.New("Check withdraw transaction failed, cross chain amount less than 0")
			}
			oriOutputAmount += common.Fixed64(float64(*w.CrossChainAmount) / exchangeRate)
			totalFee += common.Fixed64(float64(*w.Amount-*w.CrossChainAmount) / exchangeRate)
		}
		totalCrossChainCount += len(tx.WithdrawInfo.WithdrawAssets)
	}

	if inputTotalAmount != outputTotalAmount+totalFee {
		log.Info("inputTotalAmount-", inputTotalAmount, " outputTotalAmount-", outputTotalAmount, " totalFee-", totalFee)
		return errors.New("Check withdraw transaction failed, input amount not equal output amount")
	}

	//check exchange rate
	genesisBlockProgramHash, err := common.Uint168FromAddress(payloadWithdraw.GenesisBlockAddress)
	if err != nil {
		return errors.New("Check withdraw transaction failed, genesis block address to program hash failed")
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
		return errors.New("Check withdraw transaction failed, cross chain count not equal withdraw output count")
	}

	if oriOutputAmount != withdrawOutputAmount {
		log.Info("oriOutputAmount-", oriOutputAmount, " withdrawOutputAmount-", withdrawOutputAmount)
		return errors.New("Check withdraw transaction failed, exchange rate verify failed")
	}

	return nil
}
