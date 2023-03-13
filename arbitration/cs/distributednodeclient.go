package cs

import (
	"bytes"
	"errors"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
)

type DistributedNodeClient struct {
	CheckedTransactions map[common.Uint256]struct{}
}

type DistributedNodeClientFunc interface {
	GetSideChainAndExchangeRate(genesisAddress string) (arbitrator.SideChain, float64, error)
}

func (client *DistributedNodeClient) GetSideChainAndExchangeRate(genesisAddress string) (arbitrator.SideChain, float64, error) {

	sideChain, ok := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(genesisAddress)
	if !ok || sideChain == nil {
		return nil, 0, errors.New("GetSideChainAndExchangeRate Get side chain from genesis address failed.")
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

func (client *DistributedNodeClient) SignSchnorrProposal2(item *DistributedItem) error {
	return item.SchnorrSign2(arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator())
}

func (client *DistributedNodeClient) SignSchnorrProposal3(item *DistributedItem) error {
	return item.SchnorrSign3(arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator())
}

func (client *DistributedNodeClient) OnReceivedProposal(id peer.PID, content []byte) error {
	transactionItem := &DistributedItem{}
	if err := transactionItem.Deserialize(bytes.NewReader(content)); err != nil {
		return err
	}

	if err := transactionItem.CheckMyselfInCurrentArbiters(); err != nil {
		return err
	}

	switch transactionItem.Type {
	case MultisigContent:
		return client.onReceivedProposal(id, transactionItem)
	case SchnorrMultisigContent2:
		return client.onReceivedSchnorrProposal2(id, transactionItem)
	case SchnorrMultisigContent3:
		return client.onReceivedSchnorrProposal3(id, transactionItem)
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

func (client *DistributedNodeClient) onReceivedSchnorrProposal2(id peer.PID, transactionItem *DistributedItem) error {
	currentAccount := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	k0, rx, ry, px, py, err := currentAccount.GetSchnorrR()
	if err != nil {
		return err
	}
	transactionItem.SchnorrRequestRProposalContent.R = KRP{
		K0: k0,
		Rx: rx,
		Ry: ry,
		Px: px,
		Py: py,
	}
	if err := client.SignSchnorrProposal2(transactionItem); err != nil {
		return err
	}

	if err := client.Feedback(id, transactionItem); err != nil {
		return err
	}

	return nil
}

func (client *DistributedNodeClient) onReceivedSchnorrProposal3(id peer.PID, transactionItem *DistributedItem) error {
	// check if I am in public keys.
	currentAccount := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	myself, err := currentAccount.GetPublicKey().EncodePoint(true)
	if err != nil {
		return err
	}
	var needDeal bool
	for _, pk := range transactionItem.SchnorrRequestSProposalContent.Publickeys {
		if bytes.Equal(myself, pk) {
			needDeal = true
			break
		}
	}
	if !needDeal {
		return errors.New("no need to deal with the shcnorr proposal 3")
	}

	// check the transaction
	hash := transactionItem.SchnorrRequestSProposalContent.Tx.Hash()
	if _, ok := client.CheckedTransactions[hash]; !ok {
		if err := transactionItem.SchnorrRequestSProposalContent.Check(client); err != nil {
			return err
		}
		client.CheckedTransactions[hash] = struct{}{}
	}

	s := currentAccount.GetSchnorrS(transactionItem.SchnorrRequestSProposalContent.E)
	transactionItem.SchnorrRequestSProposalContent.S = s
	transactionItem.Type = AnswerSchnorrMultisigContent3

	if err := client.SignSchnorrProposal3(transactionItem); err != nil {
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

	switch item.Type {
	case MultisigContent:
		item.Type = AnswerMultisigContent

	case IllegalContent:
		item.Type = AnswerIllegalContent

	case SchnorrMultisigContent2:
		item.Type = AnswerSchnorrMultisigContent2

	case SchnorrMultisigContent3:
		item.Type = AnswerSchnorrMultisigContent3

	}

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
