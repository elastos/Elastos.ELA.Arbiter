package cs

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sync"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/common"
	"github.com/elastos/Elastos.ELA.Arbiter/common/config"
	"github.com/elastos/Elastos.ELA.Arbiter/common/log"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
	"github.com/elastos/Elastos.ELA.Arbiter/crypto"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/SPVWallet/p2p"
)

const (
	TransactionAgreementRatio = 0.667 //over 2/3 of arbitrators agree to unlock the redeem script
)

type DistributedNodeServer struct {
	mux                  *sync.Mutex
	P2pCommand           string
	unsolvedTransactions map[common.Uint256]*tx.Transaction
	finishedTransactions map[common.Uint256]bool
}

func (dns *DistributedNodeServer) tryInit() {
	if dns.mux == nil {
		dns.mux = new(sync.Mutex)
	}
	if dns.unsolvedTransactions == nil {
		dns.unsolvedTransactions = make(map[common.Uint256]*tx.Transaction)
	}
	if dns.finishedTransactions == nil {
		dns.finishedTransactions = make(map[common.Uint256]bool)
	}
}

func (dns *DistributedNodeServer) UnsolvedTransactions() map[common.Uint256]*tx.Transaction {
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
		temp := &crypto.PublicKey{}
		temp.FromString(arStr)
		publicKeys = append(publicKeys, temp)
	}
	redeemScript, err := tx.CreateWithdrawRedeemScript(getTransactionAgreementArbitratorsCount(), publicKeys)
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

func (dns *DistributedNodeServer) OnP2PReceived(peer *p2p.Peer, msg p2p.Message) error {
	if msg.CMD() != dns.P2pCommand {
		return nil
	}

	signMessage, ok := msg.(*SignMessage)
	if !ok {
		log.Warn("Unknown p2p message content.")
		return nil
	}

	return dns.ReceiveProposalFeedback(signMessage.Content)
}

func (dns *DistributedNodeServer) BroadcastWithdrawProposal(transaction *tx.Transaction) error {
	proposal, err := dns.generateWithdrawProposal(transaction)
	if err != nil {
		return err
	}

	dns.sendToArbitrator(proposal)
	return nil
}

func (dns *DistributedNodeServer) generateWithdrawProposal(transaction *tx.Transaction) ([]byte, error) {
	dns.tryInit()
	dns.mux.Lock()
	defer dns.mux.Unlock()

	if _, ok := dns.unsolvedTransactions[transaction.Hash()]; ok {
		return nil, errors.New("Transaction already in process.")
	}
	dns.unsolvedTransactions[transaction.Hash()] = transaction

	currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()
	if !currentArbitrator.IsOnDuty() {
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
	transactionItem.Sign(currentArbitrator)

	buf := new(bytes.Buffer)
	err = transactionItem.Serialize(buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (dns *DistributedNodeServer) ReceiveProposalFeedback(content []byte) error {
	dns.mux.Lock()
	defer dns.mux.Unlock()

	transactionItem := DistributedItem{}
	transactionItem.Deserialize(bytes.NewReader(content))
	newSign, err := transactionItem.ParseFeedbackSignedData()
	if err != nil {
		return err
	}

	if dns.unsolvedTransactions == nil {
		return errors.New("Can not find proposal.")
	}
	txn, ok := dns.unsolvedTransactions[transactionItem.ItemContent.Hash()]
	if !ok {
		errors.New("Can not find proposal.")
	}

	var signerIndex = -1
	programHashes, err := txn.GetMultiSignSigners()
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

	signedCount, err := dns.mergeSignToTransaction(newSign, signerIndex, txn)
	if err != nil {
		return err
	}

	if signedCount >= getTransactionAgreementArbitratorsCount() {
		delete(dns.unsolvedTransactions, txn.Hash())

		content, err := dns.convertToTransactionContent(txn)
		if err != nil {
			dns.finishedTransactions[txn.Hash()] = false
			return err
		}

		result, err := rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("Data", content), config.Parameters.MainNode.Rpc)
		if err != nil {
			return err
		}
		dns.finishedTransactions[txn.Hash()] = true
		fmt.Println(result)
	}
	return nil
}

func (dns *DistributedNodeServer) convertToTransactionContent(txn *tx.Transaction) (string, error) {
	buf := new(bytes.Buffer)
	err := txn.Serialize(buf)
	if err != nil {
		return "", err
	}
	content := common.BytesToHexString(buf.Bytes())
	return content, nil
}

func (dns *DistributedNodeServer) mergeSignToTransaction(newSign []byte, signerIndex int, txn *tx.Transaction) (int, error) {
	param := txn.Programs[0].Parameter

	// Check if is first signature
	if param == nil {
		param = []byte{}
	} else {
		// Check if singer already signed
		publicKeys, err := txn.GetMultiSignPublicKeys()
		if err != nil {
			return 0, err
		}
		buf := new(bytes.Buffer)
		txn.SerializeUnsigned(buf)
		for i := 0; i < len(param); i += tx.SignatureScriptLength {
			// Remove length byte
			sign := param[i : i+tx.SignatureScriptLength][1:]
			publicKey := publicKeys[signerIndex][1:]
			pubKey, err := crypto.DecodePoint(publicKey)
			if err != nil {
				return 0, err
			}
			err = crypto.Verify(*pubKey, buf.Bytes(), sign)
			if err == nil {
				return 0, errors.New("signer already signed")
			}
		}
	}

	buf := new(bytes.Buffer)
	buf.Write(param)
	buf.Write(newSign)

	txn.Programs[0].Parameter = buf.Bytes()
	return len(txn.Programs[0].Parameter) / tx.SignatureScriptLength, nil
}
