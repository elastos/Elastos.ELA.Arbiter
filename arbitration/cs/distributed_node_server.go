package cs

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sync"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA.Arbiter/store"
	scError "github.com/elastos/Elastos.ELA.SideChain/errors"
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
}

func (dns *DistributedNodeServer) tryInit() {
	if dns.mux == nil {
		dns.mux = new(sync.Mutex)
	}
	if dns.unsolvedTransactions == nil {
		dns.unsolvedTransactions = make(map[common.Uint256]*Transaction)
	}
}

func (dns *DistributedNodeServer) UnsolvedTransactions() map[common.Uint256]*Transaction {
	dns.mux.Lock()
	defer dns.mux.Unlock()
	return dns.unsolvedTransactions
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
	msg := &SignMessage{
		Command: dns.P2pCommand,
		Content: content,
	}
	P2PClientSingleton.AddMessageHash(P2PClientSingleton.GetMessageHash(msg))
	P2PClientSingleton.Broadcast(msg)
	log.Info("[sendToArbitrator] Send withdraw transaction to arbtiers for multi sign")
}

func (dns *DistributedNodeServer) BroadcastWithdrawProposal(transaction *Transaction) error {

	proposal, err := dns.generateWithdrawProposal(transaction, &DistrubutedItemFuncImpl{})
	if err != nil {
		return err
	}

	dns.sendToArbitrator(proposal)

	return nil
}

func (dns *DistributedNodeServer) generateWithdrawProposal(transaction *Transaction, itemFunc DistrubutedItemFunc) ([]byte, error) {
	dns.tryInit()

	currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()
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

	dns.mux.Lock()
	defer dns.mux.Unlock()

	if _, ok := dns.unsolvedTransactions[transaction.Hash()]; ok {
		return nil, errors.New("Transaction already in process.")
	}
	dns.unsolvedTransactions[transaction.Hash()] = transaction

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
	programHashes, err := crypto.GetCrossChainSigners(txn.Programs[0].Code)
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

		withdrawPayload, ok := txn.Payload.(*PayloadWithdrawFromSideChain)
		if !ok {
			return errors.New("Received proposal feed back but withdraw transaction has invalid payload")
		}

		currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()
		resp, err := currentArbitrator.SendWithdrawTransaction(txn)

		if err != nil || scError.ErrCode(resp.Code) != scError.ErrDoubleSpend &&
			scError.ErrCode(resp.Code) != scError.Success {
			log.Warn("Send withdraw transaction failed, txHash:", txn.Hash().String())

			buf := new(bytes.Buffer)
			err := txn.Serialize(buf)
			if err != nil {
				return errors.New("Send withdraw transaction faild, invalid transaction")
			}

			err = store.DbCache.RemoveSideChainTxs(withdrawPayload.SideChainTransactionHashes)
			if err != nil {
				return errors.New("Remove failed withdraw transaction from db failed")
			}
			err = store.FinishedTxsDbCache.AddWithdrawTx(withdrawPayload.SideChainTransactionHashes, buf.Bytes(), false)
			if err != nil {
				return errors.New("Add failed withdraw transaction into finished db failed")
			}
		} else if resp.Result != nil && scError.ErrCode(resp.Code) == scError.Success {
			log.Info("Send withdraw transaction succeed, txHash:", txn.Hash().String())

			err = store.DbCache.RemoveSideChainTxs(withdrawPayload.SideChainTransactionHashes)
			if err != nil {
				return errors.New("Remove succeed withdraw transaction from db failed")
			}
			err = store.FinishedTxsDbCache.AddSucceedWIthdrawTx(withdrawPayload.SideChainTransactionHashes)
			if err != nil {
				return errors.New("Add succeed withdraw transaction into finished db failed")
			}
		} else {
			log.Warn("Send withdraw transaction failed, need to resend")
		}

		fmt.Println(resp.Result)
	}
	return nil
}
