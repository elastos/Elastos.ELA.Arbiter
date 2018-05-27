package cs

import (
	"bytes"
	"errors"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	. "github.com/elastos/Elastos.ELA/core"
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

	withdrawAsset, ok := transactionItem.ItemContent.Payload.(*PayloadWithdrawAsset)
	if !ok {
		return errors.New("Unknown payload type.")
	}

	//if has one transaction not received before, then we need to sign it.
	hasSigned := true
	for _, txHash := range withdrawAsset.SideChainTransactionHash {
		ok, err := store.DbCache.HasSideChainTxProposal(txHash)
		if err != nil {
			return errors.New("Get exist side chain transaction from db failed")
		}
		if !ok {
			hasSigned = false
		}
	}

	if hasSigned {
		transactions, err := store.DbCache.GetSideChainTxsProposalFromHashes(withdrawAsset.SideChainTransactionHash)
		if err != nil {
			return errors.New("Get exist side chain transaction from db failed")
		}

		for _, txn := range transactions {
			switch txn.Payload.(type) {
			case *PayloadWithdrawAsset:
				if withdrawAsset.BlockHeight > txn.Payload.(*PayloadWithdrawAsset).BlockHeight {
					return errors.New("Proposal already exist.")
				}
			}
		}
	}

	currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()
	sc, ok := currentArbitrator.GetSideChainManager().GetChain(withdrawAsset.GenesisBlockAddress)
	if !ok {
		return errors.New("Get side chain from GenesisBlockAddress failed")
	}

	if withdrawAsset.BlockHeight > sc.GetLastUsedUtxoHeight() {
		var outPoints []OutPoint
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

	for _, txHash := range withdrawAsset.SideChainTransactionHash {
		if err := store.DbCache.AddSideChainTxProposal(txHash, withdrawAsset.GenesisBlockAddress,
			transactionItem.ItemContent); err != nil {
			return err
		}
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
