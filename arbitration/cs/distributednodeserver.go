package cs

import (
	"bytes"
	"errors"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
)

const (
	MCErrDoubleSpend          int64 = 45010
	MCErrSidechainTxDuplicate int64 = 45012
)

type DistributedNodeServer struct {
	mux              *sync.Mutex
	withdrawMux      *sync.Mutex
	unsolvedContents map[common.Uint256]base.DistributedContent
}

func (dns *DistributedNodeServer) tryInit() {
	if dns.mux == nil {
		dns.mux = new(sync.Mutex)
	}
	if dns.withdrawMux == nil {
		dns.withdrawMux = new(sync.Mutex)
	}
	if dns.unsolvedContents == nil {
		dns.unsolvedContents = make(map[common.Uint256]base.DistributedContent)
	}
}

func (dns *DistributedNodeServer) UnsolvedTransactions() map[common.Uint256]base.DistributedContent {
	dns.mux.Lock()
	defer dns.mux.Unlock()
	return dns.unsolvedContents
}

func CreateRedeemScript() ([]byte, error) {
	var publicKeys []*crypto.PublicKey
	for _, arStr := range arbitrator.ArbitratorGroupSingleton.GetAllArbitrators() {
		temp, err := base.PublicKeyFromString(arStr)
		if err != nil {
			return nil, err
		}
		publicKeys = append(publicKeys, temp)
	}
	redeemScript, err := base.CreateWithdrawRedeemScript(
		getTransactionAgreementArbitratorsCount(len(publicKeys)), publicKeys)
	if err != nil {
		return nil, err
	}
	return redeemScript, nil
}

func getTransactionAgreementArbitratorsCount(arbitersCount int) int {
	return arbitersCount*2/3 + 1
}

func (dns *DistributedNodeServer) sendToArbitrator(content []byte) {
	msg := &DistributedItemMessage{
		Content: content,
	}

	P2PClientSingleton.BroadcastMessage(msg)
	log.Info("[sendToArbitrator] Send withdraw transaction to arbiters for multi sign")
}

func (dns *DistributedNodeServer) BroadcastWithdrawProposal(txn *types.Transaction) error {

	proposal, err := dns.generateDistributedProposal(&TxDistributedContent{Tx: txn}, &DistrubutedItemFuncImpl{})
	if err != nil {
		return err
	}

	dns.sendToArbitrator(proposal)

	return nil
}

func (dns *DistributedNodeServer) BroadcastSidechainIllegalData(data *payload.SidechainIllegalData) error {

	proposal, err := dns.generateDistributedProposal(&IllegalDistributedContent{Evidence: data}, &DistrubutedItemFuncImpl{})
	if err != nil {
		return err
	}

	dns.sendToArbitrator(proposal)

	return nil
}

func (dns *DistributedNodeServer) generateDistributedProposal(itemContent base.DistributedContent, itemFunc DistrubutedItemFunc) ([]byte, error) {
	dns.tryInit()

	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	pkBuf, err := currentArbitrator.GetPublicKey().EncodePoint(true)
	if err != nil {
		return nil, err
	}
	programHash, err := contract.PublicKeyToStandardProgramHash(pkBuf)
	if err != nil {
		return nil, err
	}
	transactionItem := &DistributedItem{
		ItemContent:                 itemContent,
		TargetArbitratorPublicKey:   currentArbitrator.GetPublicKey(),
		TargetArbitratorProgramHash: programHash,
	}

	if err = transactionItem.InitScript(currentArbitrator); err != nil {
		return nil, err
	}
	if err = transactionItem.Sign(currentArbitrator, false, itemFunc); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err = transactionItem.Serialize(buf); err != nil {
		return nil, err
	}

	if err = itemContent.InitSign(transactionItem.GetSignedData()); err != nil {
		return nil, err
	}

	dns.mux.Lock()
	defer dns.mux.Unlock()

	if _, ok := dns.unsolvedContents[itemContent.Hash()]; ok {
		return nil, errors.New("transaction already in process")
	}
	dns.unsolvedContents[itemContent.Hash()] = itemContent

	return buf.Bytes(), nil
}

func (dns *DistributedNodeServer) ReceiveProposalFeedback(content []byte) error {
	dns.tryInit()
	dns.withdrawMux.Lock()
	defer dns.withdrawMux.Unlock()

	transactionItem := DistributedItem{}
	if err := transactionItem.Deserialize(bytes.NewReader(content)); err != nil {
		return err
	}
	newSign, err := transactionItem.ParseFeedbackSignedData()
	if err != nil {
		return err
	}

	dns.mux.Lock()
	if dns.unsolvedContents == nil {
		dns.mux.Unlock()
		return errors.New("can not find proposal")
	}
	txn, ok := dns.unsolvedContents[transactionItem.ItemContent.Hash()]
	if !ok {
		dns.mux.Unlock()
		return errors.New("can not find proposal")
	}
	dns.mux.Unlock()

	targetCodeHash := transactionItem.TargetArbitratorProgramHash.ToCodeHash()
	signedCount, err := txn.MergeSign(newSign, &targetCodeHash)
	if err != nil {
		return err
	}

	if signedCount >= getTransactionAgreementArbitratorsCount(len(arbitrator.ArbitratorGroupSingleton.GetAllArbitrators())) {
		dns.mux.Lock()
		delete(dns.unsolvedContents, txn.Hash())
		dns.mux.Unlock()

		if err = txn.Submit(); err != nil {
			log.Warn(err.Error())
			return err
		}
	}
	return nil
}
