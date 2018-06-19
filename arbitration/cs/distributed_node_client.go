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

func (client *DistributedNodeClient) SignProposal(item *DistributedItem) error {
	return item.Sign(ArbitratorGroupSingleton.GetCurrentArbitrator(), true, &DistrubutedItemFuncImpl{})
}

func (client *DistributedNodeClient) OnReceivedProposal(content []byte) error {
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

	err := checkWithdrawTransaction(transactionItem.ItemContent)
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

func checkWithdrawTransaction(tx *ela.Transaction) error {
	payloadWithdraw, ok := tx.Payload.(*ela.PayloadWithdrawFromSideChain)
	if !ok {
		return errors.New("Check withdraw transaction failed, unknown payload type")
	}

	sideChain, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(payloadWithdraw.GenesisBlockAddress)
	if !ok {
		return errors.New("Check withdraw transaction failed, unknown genesis block address")
	}

	//check genesis address
	var txs []*ela.Transaction
	sideChainTxs, err := store.DbCache.SideChainStore.GetSideChainTxsFromHashesAndGenesisAddress(
		payloadWithdraw.SideChainTransactionHashes, payloadWithdraw.GenesisBlockAddress)
	if err != nil || len(sideChainTxs) != len(payloadWithdraw.SideChainTransactionHashes) {
		log.Info("Check withdraw transaction, need to get side chain transaction from rpc")
		for _, txHash := range payloadWithdraw.SideChainTransactionHashes {
			tx, err := sideChain.GetTransactionByHash(txHash)
			if err != nil {
				return errors.New("Check withdraw transaction failed, unknown side chain transachtions")
			}
			txs = append(txs, tx)
		}
	} else {
		txs = sideChainTxs
	}

	utxos, err := store.DbCache.UTXOStore.GetAddressUTXOsFromGenesisBlockAddress(payloadWithdraw.GenesisBlockAddress)
	if err != nil {
		return errors.New("Get spender's UTXOs failed.")
	}

	//check inputs
	var inputTotalAmount common.Fixed64
	for _, input := range tx.Inputs {
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
	rate := common.Fixed64(sideChain.GetExchangeRate())

	var outputTotalAmount common.Fixed64
	for _, output := range tx.Outputs {
		outputTotalAmount += output.Value
	}

	var totalFee common.Fixed64
	var oriOutputAmount common.Fixed64
	for _, tx := range txs {
		payloadObj, ok := tx.Payload.(*ela.PayloadTransferCrossChainAsset)
		if !ok {
			return errors.New("Check withdraw transaction failed, invalid side chain transaction payload")
		}
		for _, amount := range payloadObj.CrossChainAmount {
			oriOutputAmount += amount / rate
		}
		for i := 0; i < len(payloadObj.CrossChainAddress); i++ {
			totalFee += (tx.Outputs[payloadObj.OutputIndex[i]].Value - payloadObj.CrossChainAmount[i]) / rate
		}
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
	for _, output := range tx.Outputs {
		if output.ProgramHash != *genesisBlockProgramHash {
			withdrawOutputAmount += output.Value
		}
	}

	if oriOutputAmount != withdrawOutputAmount {
		log.Info("oriOutputAmount-", oriOutputAmount, " withdrawOutputAmount-", withdrawOutputAmount)
		return errors.New("Check withdraw transaction failed, exchange rate verify failed")
	}

	return nil
}
