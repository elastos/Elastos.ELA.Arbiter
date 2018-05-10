package cs

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sync"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
	. "github.com/elastos/Elastos.ELA/core"
)

const (
	TransactionAgreementRatio = 0.667 //over 2/3 of arbitrators agree to unlock the redeem script
)

type DistributedNodeServer struct {
	mux                  *sync.Mutex
	P2pCommand           string
	unsolvedTransactions map[common.Uint256]*Transaction
	finishedTransactions map[common.Uint256]bool
}

func (dns *DistributedNodeServer) tryInit() {
	if dns.mux == nil {
		dns.mux = new(sync.Mutex)
	}
	if dns.unsolvedTransactions == nil {
		dns.unsolvedTransactions = make(map[common.Uint256]*Transaction)
	}
	if dns.finishedTransactions == nil {
		dns.finishedTransactions = make(map[common.Uint256]bool)
	}
}

func (dns *DistributedNodeServer) UnsolvedTransactions() map[common.Uint256]*Transaction {
	dns.mux.Lock()
	defer dns.mux.Unlock()
	return dns.unsolvedTransactions
}

func (dns *DistributedNodeServer) FinishedTransactions() map[common.Uint256]bool {
	dns.mux.Lock()
	defer dns.mux.Unlock()
	return dns.finishedTransactions
}

func CreateRedeemScript() ([]byte, error) {
	var publicKeys []*crypto.PublicKey
	for _, arStr := range ArbitratorGroupSingleton.GetAllArbitrators() {
		temp, err := PublicKeyFromString(arStr)
		if err != nil {
			return nil, err
		}
		publicKeys = append(publicKeys, temp)
	}
	redeemScript, err := CreateWithdrawRedeemScript(
		getTransactionAgreementArbitratorsCount(), publicKeys)
	if err != nil {
		return nil, err
	}
	return redeemScript, nil
}

func getTransactionAgreementArbitratorsCount() int {
	return int(math.Ceil(float64(ArbitratorGroupSingleton.GetArbitratorsCount()) * TransactionAgreementRatio))
}

func (dns *DistributedNodeServer) sendToArbitrator(content []byte) {
	P2PClientSingleton.Broadcast(&SignMessage{
		Command: dns.P2pCommand,
		Content: content,
	})
}

func (dns *DistributedNodeServer) BroadcastWithdrawProposal(transaction *Transaction) error {

	withdrawAsset, ok := transaction.Payload.(*PayloadWithdrawAsset)
	if !ok {
		return errors.New("Unknown playload typed.")
	}

	proposal, err := dns.generateWithdrawProposal(transaction, &DistrubutedItemFuncImpl{})
	if err != nil {
		return err
	}

	dns.sendToArbitrator(proposal)

	if err := store.DbCache.AddSideChainTx(
		withdrawAsset.SideChainTransactionHash, withdrawAsset.GenesisBlockAddress); err != nil {
		return err
	}

	return nil
}

func (dns *DistributedNodeServer) generateWithdrawProposal(transaction *Transaction, itemFunc DistrubutedItemFunc) ([]byte, error) {
	dns.tryInit()

	dns.mux.Lock()
	if _, ok := dns.unsolvedTransactions[transaction.Hash()]; ok {
		return nil, errors.New("Transaction already in process.")
	}
	dns.unsolvedTransactions[transaction.Hash()] = transaction
	dns.mux.Unlock()

	currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()
	if !currentArbitrator.IsOnDutyOfMain() {
		return nil, errors.New("Can not start a new proposal, you are not on duty.")
	}

	programHash, err := StandardAcccountPublicKeyToProgramHash(currentArbitrator.GetPublicKey())
	if err != nil {
		return nil, err
	}
	transactionItem := &DistributedItem{
		ItemContent:                 transaction,
		TargetArbitratorPublicKey:   currentArbitrator.GetPublicKey(),
		TargetArbitratorProgramHash: programHash,
	}
	transactionItem.InitScript(currentArbitrator)
	transactionItem.Sign(currentArbitrator, false, itemFunc)

	buf := new(bytes.Buffer)
	err = transactionItem.Serialize(buf)
	if err != nil {
		return nil, err
	}

	transaction.Programs[0].Parameter = transactionItem.GetSignedData()

	return buf.Bytes(), nil
}

func (dns *DistributedNodeServer) ReceiveProposalFeedback(content []byte) error {
	dns.tryInit()

	transactionItem := DistributedItem{}
	transactionItem.Deserialize(bytes.NewReader(content))
	newSign, err := transactionItem.ParseFeedbackSignedData()
	if err != nil {
		return err
	}

	dns.mux.Lock()
	if dns.unsolvedTransactions == nil {
		dns.mux.Unlock()
		return errors.New("Can not find proposal.")
	}
	txn, ok := dns.unsolvedTransactions[transactionItem.ItemContent.Hash()]
	if !ok {
		dns.mux.Unlock()
		return errors.New("Can not find proposal.")
	}
	dns.mux.Unlock()

	var signerIndex = -1
	programHashes, err := crypto.GetSigners(txn.Programs[0].Code)
	if err != nil {
		return err
	}
	userProgramHash := transactionItem.TargetArbitratorProgramHash
	for i, programHash := range programHashes {
		if *userProgramHash == *programHash {
			signerIndex = i
			break
		}
	}
	if signerIndex == -1 {
		return errors.New("Invalid multi sign signer")
	}

	signedCount, err := MergeSignToTransaction(newSign, signerIndex, txn)
	if err != nil {
		return err
	}

	if signedCount >= getTransactionAgreementArbitratorsCount() {
		dns.mux.Lock()
		delete(dns.unsolvedTransactions, txn.Hash())
		dns.mux.Unlock()

		currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()
		result, err := currentArbitrator.SendWithdrawTransaction(txn)

		if err != nil {
			dns.mux.Lock()
			dns.finishedTransactions[txn.Hash()] = false
			dns.mux.Unlock()
			return err
		}

		dns.mux.Lock()
		dns.finishedTransactions[txn.Hash()] = true
		dns.mux.Unlock()

		fmt.Println(result)
	}
	return nil
}
