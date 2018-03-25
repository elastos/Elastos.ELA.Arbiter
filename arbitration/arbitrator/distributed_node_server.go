package arbitrator

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/common/config"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/crypto"
	"Elastos.ELA.Arbiter/rpc"
)

const (
	TransactionAgreementRatio = 0.667 //over 2/3 of arbitrators agree to unlock the redeem script
)

type DistributedNodeServer struct {
	unsolvedTransactions map[common.Uint256]*tx.Transaction
	finishedTransactions map[common.Uint256]bool
}

func (dns *DistributedNodeServer) UnsolvedTransactions() map[common.Uint256]*tx.Transaction {
	return dns.unsolvedTransactions
}

func (dns *DistributedNodeServer) FinishedTransactions() map[common.Uint256]bool {
	return dns.finishedTransactions
}

func (dns *DistributedNodeServer) CreateRedeemScript() ([]byte, error) {
	arbitratorCount := ArbitratorGroupSingleton.GetArbitratorsCount()
	publicKeys := make([]*crypto.PublicKey, arbitratorCount)
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

func (dns *DistributedNodeServer) sendToArbitrator(otherArbitrator string, content []byte) error {
	//todo call p2p module to broadcast to other arbitrators
	return nil
}

func (dns *DistributedNodeServer) BroadcastWithdrawProposal(transaction *tx.Transaction) error {
	proposals, err := dns.generateWithdrawProposals(transaction)
	if err != nil {
		return err
	}

	for pkStr, content := range proposals {
		dns.sendToArbitrator(pkStr, content)
	}
	return nil
}

func (dns *DistributedNodeServer) generateWithdrawProposals(transaction *tx.Transaction) (map[string][]byte, error) {
	if _, ok := dns.unsolvedTransactions[transaction.Hash()]; ok {
		return nil, errors.New("Transaction already in process.")
	}
	dns.unsolvedTransactions[transaction.Hash()] = transaction

	currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()
	if !currentArbitrator.IsOnDuty() {
		return nil, errors.New("Can not start a new proposal, you are not on duty.")
	}

	publicKeys := make(map[string]*crypto.PublicKey, ArbitratorGroupSingleton.GetArbitratorsCount())
	for _, arStr := range ArbitratorGroupSingleton.GetAllArbitrators() {
		temp := &crypto.PublicKey{}
		temp.FromString(arStr)
		publicKeys[arStr] = temp
	}

	results := make(map[string][]byte, len(publicKeys))
	for pkStr, pk := range publicKeys {
		programHash, err := StandardAcccountPublicKeyToProgramHash(pk)
		if err != nil {
			return nil, err
		}
		transactionItem := &DistributedItem{
			ItemContent:                 transaction,
			TargetArbitratorPublicKey:   pk,
			TargetArbitratorProgramHash: programHash,
		}
		transactionItem.InitScript(currentArbitrator)
		transactionItem.Sign(currentArbitrator)

		buf := new(bytes.Buffer)
		err = transactionItem.Serialize(buf)
		if err != nil {
			return nil, err
		}
		results[pkStr] = buf.Bytes()
	}
	return results, nil
}

func (dns *DistributedNodeServer) ReceiveProposalFeedback(content []byte) error {
	transactionItem := DistributedItem{}
	transactionItem.Deserialize(bytes.NewReader(content))
	newSign, err := transactionItem.ParseFeedbackSignedData()
	if err != nil {
		return err
	}

	trans, ok := transactionItem.ItemContent.(*tx.Transaction)
	if !ok {
		return errors.New("Unknown transaction content.")
	}
	txn, ok := dns.unsolvedTransactions[trans.Hash()]
	if !ok {
		errors.New("Can not find transaction.")
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
