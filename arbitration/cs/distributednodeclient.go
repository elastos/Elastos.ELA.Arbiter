package cs

import (
	"bytes"
	"errors"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"

	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
)

type DistributedNodeClient struct {
}

type DistributedNodeClientFunc interface {
	GetSideChainAndExchangeRate(genesisAddress string) (arbitrator.SideChain, float64, error)
}

func (client *DistributedNodeClient) GetSideChainAndExchangeRate(genesisAddress string) (arbitrator.SideChain, float64, error) {
	sideChain, ok := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(genesisAddress)
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
	return item.Sign(arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator(), true, &DistrubutedItemFuncImpl{})
}

func (client *DistributedNodeClient) SignSchnorrProposal1(item *DistributedItem) error {
	return item.SchnorrSign1(arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator())
}

func (client *DistributedNodeClient) OnReceivedProposal(id peer.PID, content []byte) error {
	transactionItem := &DistributedItem{}
	if err := transactionItem.Deserialize(bytes.NewReader(content)); err != nil {
		return err
	}

	switch transactionItem.Type {
	case TxDistribute, IllegalDistribute, ReturnDepositDistribute:
		return client.onReceivedProposal(id, transactionItem)
	case SchnorrWithdrawProposal:
		return client.onReceivedSchnorrProposal1(id, transactionItem)
	}
	return nil
}

func (client *DistributedNodeClient) onReceivedProposal(id peer.PID, transactionItem *DistributedItem) error {
	if err := transactionItem.ItemContent.Check(client); err != nil {
		return err
	}

	if err := client.SignProposal(transactionItem); err != nil {
		return err
	}

	if err := client.Feedback(id, transactionItem); err != nil {
		return err
	}

	return nil
}

func (client *DistributedNodeClient) onReceivedSchnorrProposal1(id peer.PID, transactionItem *DistributedItem) error {
	if len(transactionItem.signedData) == 0 {
		return nil
	}

	if err := transactionItem.SchnorrProposalContent.Check(client); err != nil {
		return err
	}

	if err := client.SignSchnorrProposal1(transactionItem); err != nil {
		return err
	}

	if err := client.Feedback(id, transactionItem); err != nil {
		return err
	}

	return nil
}

func (client *DistributedNodeClient) Feedback(id peer.PID, item *DistributedItem) error {
	ar := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	item.TargetArbitratorPublicKey = ar.GetPublicKey()

	pkBuf, err := item.TargetArbitratorPublicKey.EncodePoint(true)
	if err != nil {
		return err
	}
	programHash, err := contract.PublicKeyToStandardProgramHash(pkBuf)
	if err != nil {
		return err
	}
	item.TargetArbitratorProgramHash = programHash

	messageReader := new(bytes.Buffer)
	err = item.Serialize(messageReader)
	if err != nil {
		return errors.New("Send complaint failed.")
	}

	return P2PClientSingleton.SendMessageToPeer(id, &DistributedItemMessage{
		Content: messageReader.Bytes(),
	})
}
