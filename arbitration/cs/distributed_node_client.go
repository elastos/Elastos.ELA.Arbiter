package cs

import (
	"bytes"
	"errors"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
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

	ok, err := store.DbCache.HasSideChainTxReceived(withdrawAsset.SideChainTransactionHash)
	if err != nil {
		return errors.New("Get exist side chain transaction from db failed")
	}
	if ok {
		transactions, err := store.DbCache.GetSideChainTxsFromHashes([]string{withdrawAsset.SideChainTransactionHash})
		if err != nil || len(transactions) != 1 {
			return errors.New("Get exist side chain transaction from db failed")
		}

		withdrawAsset, ok := transactions[0].Payload.(*PayloadWithdrawAsset)
		if !ok {
			return errors.New("Unknown payload type")
		}

		if withdrawAsset.BlockHeight >= transactions[0].Payload.(*PayloadWithdrawAsset).BlockHeight {
			return errors.New("Proposal already exit.")
		}
	}

	if err := client.SignProposal(transactionItem); err != nil {
		return err
	}

	if err := client.Feedback(transactionItem); err != nil {
		return err
	}

	ok, err = store.DbCache.HasSideChainTx(withdrawAsset.SideChainTransactionHash)
	if err != nil {
		return err
	}
	if !ok {
		if err := store.DbCache.AddSideChainTx(withdrawAsset.SideChainTransactionHash,
			withdrawAsset.GenesisBlockAddress, transactionItem.ItemContent, true); err != nil {
			return err
		}
	} else {
		if err := store.DbCache.RemoveSideChainTxs([]string{withdrawAsset.SideChainTransactionHash}); err != nil {
			return err
		}

		if err = store.DbCache.AddSideChainTx(withdrawAsset.SideChainTransactionHash,
			withdrawAsset.GenesisBlockAddress, transactionItem.ItemContent, true); err != nil {
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

func (client *DistributedNodeClient) Verify(withdrawAsset *PayloadWithdrawAsset) error {
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
