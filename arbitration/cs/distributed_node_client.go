package cs

import (
	"bytes"
	"errors"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/common/config"
	"github.com/elastos/Elastos.ELA.Arbiter/common/log"
	"github.com/elastos/Elastos.ELA.Arbiter/core/transaction/payload"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA.SPV/p2p"
)

type DistributedNodeClient struct {
	P2pCommand string
}

func (client *DistributedNodeClient) SignProposal(item *DistributedItem) error {
	return item.Sign(ArbitratorGroupSingleton.GetCurrentArbitrator(), true)
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

	withdrawAsset, ok := transactionItem.ItemContent.Payload.(*payload.WithdrawAsset)
	if !ok {
		return errors.New("Unknown payload type.")
	}

	if ok, err := store.DbCache.HashSideChainTx(withdrawAsset.SideChainTransactionHash); err != nil || ok {
		return errors.New("Proposal already exit.")
	}

	if err := client.SignProposal(transactionItem); err != nil {
		return err
	}

	if err := client.Feedback(transactionItem); err != nil {
		return err
	}

	if err := store.DbCache.AddSideChainTx(
		withdrawAsset.SideChainTransactionHash, withdrawAsset.GenesisBlockAddress); err != nil {
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

func (client *DistributedNodeClient) Verify(withdrawAsset *payload.WithdrawAsset) error {
	rpcConfig, ok := config.GetRpcConfig(withdrawAsset.GenesisBlockAddress)
	if !ok {
		return errors.New("Unknown side chain.")
	}

	ok, err := rpc.IsTransactionExist(withdrawAsset.SideChainTransactionHash, rpcConfig)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("Unknown transaction.")
	}

	return nil
}

func (client *DistributedNodeClient) broadcast(message []byte) {
	P2PClientSingleton.Broadcast(&SignMessage{
		Command: client.P2pCommand,
		Content: message,
	})
}
