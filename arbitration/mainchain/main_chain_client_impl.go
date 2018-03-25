package mainchain

import (
	. "Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "Elastos.ELA.Arbiter/arbitration/base"
	. "Elastos.ELA.Arbiter/common"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"bytes"
	"errors"
)

type MainChainClientImpl struct {
	unsolvedProposals map[Uint256]*DistributedItem
}

func (client *MainChainClientImpl) SignProposal(transactionHash Uint256) error {
	transactionItem, ok := client.unsolvedProposals[transactionHash]
	if !ok {
		return errors.New("Can not find proposal.")
	}

	return transactionItem.Sign(ArbitratorGroupSingleton.GetCurrentArbitrator())
}

//todo called by p2p module
func (client *MainChainClientImpl) OnReceivedProposal(content []byte) error {
	transactionItem := &DistributedItem{}
	if err := transactionItem.Deserialize(bytes.NewReader(content)); err != nil {
		return err
	}

	trans, ok := transactionItem.ItemContent.(*tx.Transaction)
	if !ok {
		return errors.New("Unknown transaction content.")
	}
	if _, ok := client.unsolvedProposals[trans.Hash()]; ok {
		return errors.New("Proposal already exit.")
	}

	client.unsolvedProposals[trans.Hash()] = transactionItem

	if err := client.SignProposal(trans.Hash()); err != nil {
		return err
	}

	if err := client.Feedback(trans.Hash()); err != nil {
		return err
	}
	return nil
}

func (client *MainChainClientImpl) Feedback(transactionHash Uint256) error {
	item, ok := client.unsolvedProposals[transactionHash]
	if !ok {
		return errors.New("Can not find proposal.")
	}

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

	return client.sendBack(messageReader.Bytes())
}

func (comp *MainChainClientImpl) sendBack(message []byte) error {
	//todo send feedback by p2p module
	return nil
}
