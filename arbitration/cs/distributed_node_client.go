package cs

import (
	"bytes"
	"errors"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	. "github.com/elastos/Elastos.ELA.Arbiter/common"
	"github.com/elastos/Elastos.ELA.Arbiter/common/log"
	"github.com/elastos/Elastos.ELA.SPV/p2p"
)

type DistributedNodeClient struct {
	P2pCommand        string
	unsolvedProposals map[Uint256]*DistributedItem
}

func (client *DistributedNodeClient) tryInit() {
	if client.unsolvedProposals == nil {
		client.unsolvedProposals = make(map[Uint256]*DistributedItem)
	}
}

func (client *DistributedNodeClient) SignProposal(transactionHash Uint256) error {
	transactionItem, ok := client.unsolvedProposals[transactionHash]
	if !ok {
		return errors.New("Can not find proposal.")
	}

	return transactionItem.Sign(ArbitratorGroupSingleton.GetCurrentArbitrator())
}

func (client *DistributedNodeClient) OnP2PReceived(peer *p2p.Peer, msg p2p.Message) error {
	if msg.CMD() != client.P2pCommand {
		return nil
	}

	signMessage, ok := msg.(*SignMessage)
	if !ok {
		log.Warn("Unknown p2p message content.")
		return nil
	}

	return client.OnReceivedProposal(signMessage.Content)
}

func (client *DistributedNodeClient) OnReceivedProposal(content []byte) error {
	transactionItem := &DistributedItem{}
	if err := transactionItem.Deserialize(bytes.NewReader(content)); err != nil {
		return err
	}

	if transactionItem.IsFeedback() {
		client.broadcast(content)
		return nil
	}

	if client.unsolvedProposals == nil {
		return errors.New("Can not find proposal.")
	}

	hash := transactionItem.ItemContent.Hash()
	if _, ok := client.unsolvedProposals[hash]; ok {
		return errors.New("Proposal already exit.")
	}

	client.tryInit()
	client.unsolvedProposals[hash] = transactionItem

	if err := client.SignProposal(hash); err != nil {
		return err
	}

	if err := client.Feedback(hash); err != nil {
		return err
	}
	return nil
}

func (client *DistributedNodeClient) Feedback(transactionHash Uint256) error {
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

	client.broadcast(messageReader.Bytes())
	return nil
}

func (client *DistributedNodeClient) broadcast(message []byte) {
	P2PClientSingleton.Broadcast(&SignMessage{
		Command: client.P2pCommand,
		Content: message,
	})
}
