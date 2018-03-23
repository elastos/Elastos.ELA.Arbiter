package mainchain

import (
	"Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "Elastos.ELA.Arbiter/arbitration/base"
	. "Elastos.ELA.Arbiter/common"
	"errors"
)

type MainChainClientImpl struct {
	unsolvedProposals map[Uint256]*DistributedTransactionItem
}

func (client *MainChainClientImpl) SignProposal(transactionHash Uint256) error {
	transactionItem, ok := client.unsolvedProposals[transactionHash]
	if !ok {
		return errors.New("Can not find proposal.")
	}

	return transactionItem.Sign(arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator())
}

//todo called by p2p module
func (client *MainChainClientImpl) OnReceivedProposal(content []byte) error {
	transactionItem := &DistributedTransactionItem{}
	if err := transactionItem.Deserialize(content); err != nil {
		return err
	}

	if _, ok := client.unsolvedProposals[transactionItem.RawTransaction.Hash()]; ok {
		return errors.New("Proposal already exit.")
	}

	client.unsolvedProposals[transactionItem.RawTransaction.Hash()] = transactionItem

	if err := client.SignProposal(transactionItem.RawTransaction.Hash()); err != nil {
		return err
	}

	if err := client.Feedback(transactionItem.RawTransaction.Hash()); err != nil {
		return err
	}
	return nil
}

func (client *MainChainClientImpl) Feedback(transactionHash Uint256) error {
	item, ok := client.unsolvedProposals[transactionHash]
	if !ok {
		return errors.New("Can not find proposal.")
	}

	ar := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	item.TargetArbitratorPublicKey = ar.GetPublicKey()

	programHash, err := StandardAcccountPublicKeyToProgramHash(item.TargetArbitratorPublicKey)
	if err != nil {
		return err
	}
	item.TargetArbitratorProgramHash = programHash

	message, err := item.Serialize()
	if err != nil {
		return errors.New("Send complaint failed.")
	}

	return client.sendBack(message)
}

func (comp *MainChainClientImpl) sendBack(message []byte) error {
	//todo send feedback by p2p module
	return nil
}
