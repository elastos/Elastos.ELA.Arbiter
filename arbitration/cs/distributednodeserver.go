package cs

import (
	"bytes"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
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
	mux                       *sync.Mutex
	withdrawMux               *sync.Mutex
	unsolvedContents          map[common.Uint256]base.DistributedContent
	unsolvedContentsSignature map[common.Uint256]map[common.Uint160]struct{}

	// schnorr withdraw
	schnorrWithdrawContentsTransaction map[common.Uint256]types.Transaction // key: nonce hash
	schnorrWithdrawContentsSigners     map[common.Uint256]map[string]struct{}
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
	if dns.unsolvedContentsSignature == nil {
		dns.unsolvedContentsSignature = make(map[common.Uint256]map[common.Uint160]struct{})
	}
	if dns.schnorrWithdrawContentsTransaction == nil {
		dns.schnorrWithdrawContentsTransaction = make(map[common.Uint256]types.Transaction)
	}
	if dns.schnorrWithdrawContentsSigners == nil {
		dns.schnorrWithdrawContentsSigners = make(map[common.Uint256]map[string]struct{})
	}
}

func (dns *DistributedNodeServer) UnsolvedTransactions() map[common.Uint256]base.DistributedContent {
	dns.mux.Lock()
	defer dns.mux.Unlock()
	return dns.unsolvedContents
}

func CreateRedeemScript() ([]byte, error) {
	var publicKeys []*crypto.PublicKey
	arbiters := arbitrator.ArbitratorGroupSingleton.GetAllArbitrators()
	for _, arStr := range arbiters {
		if arStr == "" {
			continue
		}
		temp, err := base.PublicKeyFromString(arStr)
		if err != nil {
			return nil, err
		}
		publicKeys = append(publicKeys, temp)
	}
	arbitersCount := getTransactionAgreementArbitratorsCount(len(arbiters))
	redeemScript, err := base.CreateWithdrawRedeemScript(arbitersCount, publicKeys)
	if err != nil {
		return nil, err
	}
	return redeemScript, nil
}

func CreateSchnonrrRedeemScript() ([]byte, error) {
	// todo complete me
	return nil, nil
}

func getTransactionAgreementArbitratorsCount(arbitersCount int) int {
	currentHeight := arbitrator.ArbitratorGroupSingleton.GetCurrentHeight()
	if currentHeight <= config.Parameters.CRClaimDPOSNodeStartHeight {
		return arbitersCount*2/3 + 1
	} else if currentHeight < config.Parameters.DPOSNodeCrossChainHeight {
		return arbitersCount * 2 / 3
	}
	return arbitersCount*2/3 + 1
}

func (dns *DistributedNodeServer) sendToArbitrator(content []byte) {
	msg := &DistributedItemMessage{
		Content: content,
	}

	P2PClientSingleton.BroadcastMessage(msg)
	log.Info("[sendToArbitrator] Send withdraw transaction to arbiters for multi sign")
}

func (dns *DistributedNodeServer) BroadcastSchnorrWithdrawProposal1(txn *types.Transaction) error {
	var txType TransactionType
	switch txn.TxType {
	case types.WithdrawFromSideChain:
		txType = WithdrawTransaction
	case types.ReturnCRDepositCoin:
		txType = ReturnDepositTransaction
	}

	proposal, err := dns.generateDistributedSchnorrProposal1(
		txn,
		txType, SchnorrMultisigContent1,
		SchnorrWithdrawProposalContent{
			Nonce: txn.Hash().Bytes()})
	if err != nil {
		return err
	}

	dns.sendToArbitrator(proposal)
	return nil
}

func (dns *DistributedNodeServer) BroadcastSchnorrWithdrawProposal2(txn *types.Transaction, pks [][]byte) error {
	var txType TransactionType
	switch txn.TxType {
	case types.WithdrawFromSideChain:
		txType = WithdrawTransaction
	case types.ReturnCRDepositCoin:
		txType = ReturnDepositTransaction
	}

	proposal, err := dns.generateDistributedSchnorrProposal2(txn, txType, SchnorrMultisigContent2,
		SchnorrWithdrawRequestRProposalContent{Tx: txn, Publickeys: pks})
	if err != nil {
		return err
	}

	dns.sendToArbitrator(proposal)

	return nil
}

func (dns *DistributedNodeServer) BroadcastSchnorrWithdrawProposal3(txn *types.Transaction) error {
	// todo complete me
	return nil
}

func (dns *DistributedNodeServer) BroadcastWithdrawProposal(txn *types.Transaction) error {

	var txType TransactionType
	switch txn.TxType {
	case types.WithdrawFromSideChain:
		txType = WithdrawTransaction
	case types.ReturnCRDepositCoin:
		txType = ReturnDepositTransaction
	}
	proposal, err := dns.generateDistributedProposal(txType, MultisigContent,
		&TxDistributedContent{Tx: txn}, &DistrubutedItemFuncImpl{})
	if err != nil {
		return err
	}

	dns.sendToArbitrator(proposal)

	return nil
}

func (dns *DistributedNodeServer) BroadcastSidechainIllegalData(data *payload.SidechainIllegalData) error {

	proposal, err := dns.generateDistributedProposal(IllegalTransaction, IllegalContent,
		&IllegalDistributedContent{Evidence: data}, &DistrubutedItemFuncImpl{})
	if err != nil {
		return err
	}

	dns.sendToArbitrator(proposal)

	return nil
}

func (dns *DistributedNodeServer) generateDistributedSchnorrProposal1(
	txn *types.Transaction, txType TransactionType,
	cType DistributeContentType, content SchnorrWithdrawProposalContent) ([]byte, error) {

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
		TargetArbitratorPublicKey:   currentArbitrator.GetPublicKey(),
		TargetArbitratorProgramHash: programHash,
		TransactionType:             txType,
		Type:                        cType,
		SchnorrProposalContent:      content,
	}

	buf := new(bytes.Buffer)
	if err = transactionItem.Serialize(buf); err != nil {
		return nil, err
	}

	dns.mux.Lock()
	defer dns.mux.Unlock()

	if _, ok := dns.schnorrWithdrawContentsTransaction[content.Hash()]; ok {
		return nil, errors.New("transaction already in process")
	}
	dns.schnorrWithdrawContentsTransaction[content.Hash()] = *txn
	dns.schnorrWithdrawContentsSigners[content.Hash()] = make(map[string]struct{})
	return nil, nil
}

func (dns *DistributedNodeServer) generateDistributedSchnorrProposal2(
	txn *types.Transaction, txType TransactionType,
	cType DistributeContentType, content SchnorrWithdrawRequestRProposalContent) ([]byte, error) {
	//  todo finish me
	return nil, nil
}

func (dns *DistributedNodeServer) generateDistributedSchnorrProposal3(
	txn *types.Transaction, txType TransactionType,
	cType DistributeContentType, content SchnorrWithdrawRequestRProposalContent) ([]byte, error) {
	//  todo finish me
	return nil, nil
}

func (dns *DistributedNodeServer) generateDistributedProposal(
	txType TransactionType, cType DistributeContentType,
	itemContent base.DistributedContent,
	itemFunc DistrubutedItemFunc) ([]byte, error) {
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
		TransactionType:             txType,
		Type:                        cType,
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

	signs := make(map[common.Uint160]struct{})
	signs[programHash.ToCodeHash()] = struct{}{}
	dns.unsolvedContentsSignature[itemContent.Hash()] = signs

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

	switch transactionItem.Type {
	case AnswerMultisigContent:
		return dns.receiveWithdrawProposalFeedback(transactionItem)
	case AnswerIllegalContent:
	case AnswerSchnorrMultisigContent1:
		return dns.receiveSchnorrWithdrawProposal1Feedback(transactionItem)
	case AnswerSchnorrMultisigContent2:
		return dns.receiveSchnorrWithdrawProposal1Feedback(transactionItem)
	case AnswerSchnorrMultisigContent3:
		return dns.receiveSchnorrWithdrawProposal1Feedback(transactionItem)
	}

	return nil
}

func (dns *DistributedNodeServer) receiveWithdrawProposalFeedback(transactionItem DistributedItem) error {
	newSign, msg, err := transactionItem.ParseFeedbackSignedData()
	if err != nil {
		return err
	}
	if msg != "" {
		log.Warn(msg)
		return nil
	}

	dns.mux.Lock()
	if dns.unsolvedContents == nil {
		dns.mux.Unlock()
		return errors.New("can not find proposal")
	}
	hash := transactionItem.ItemContent.Hash()
	txn, ok := dns.unsolvedContents[hash]
	if !ok {
		dns.mux.Unlock()
		return errors.New("can not find proposal")
	}
	dns.mux.Unlock()

	targetCodeHash := transactionItem.TargetArbitratorProgramHash.ToCodeHash()

	signs := dns.unsolvedContentsSignature[hash]
	if _, ok := signs[targetCodeHash]; ok {
		log.Warn("arbiter already signed ")
		return nil
	}

	signedCount, err := txn.MergeSign(newSign, &targetCodeHash)
	if err != nil {
		return err
	}
	dns.unsolvedContentsSignature[hash][targetCodeHash] = struct{}{}

	pk, _ := transactionItem.TargetArbitratorPublicKey.EncodePoint(true)
	log.Info("receive signature from ", hex.EncodeToString(pk))
	if signedCount >= getTransactionAgreementArbitratorsCount(
		len(arbitrator.ArbitratorGroupSingleton.GetAllArbitrators())) {
		dns.mux.Lock()
		delete(dns.unsolvedContents, hash)
		delete(dns.unsolvedContentsSignature, hash)
		dns.mux.Unlock()

		if err = txn.Submit(); err != nil {
			log.Warn(err.Error())
			return err
		}
	}
	return nil
}

func (dns *DistributedNodeServer) receiveSchnorrWithdrawProposal1Feedback(transactionItem DistributedItem) error {
	if err := transactionItem.CheckFeedbackSignedData(); err != nil {
		return err
	}

	dns.mux.Lock()
	if dns.schnorrWithdrawContentsTransaction == nil {
		dns.mux.Unlock()
		return errors.New("can not find proposal")
	}
	hash := transactionItem.SchnorrProposalContent.Hash()
	txn, ok := dns.schnorrWithdrawContentsTransaction[hash]
	if !ok {
		dns.mux.Unlock()
		return errors.New("can not find proposal transaction")
	}
	signers, ok := dns.schnorrWithdrawContentsSigners[hash]
	if !ok {
		dns.mux.Unlock()
		return errors.New("can not find signer")
	}
	publickeyBytes, err := transactionItem.TargetArbitratorPublicKey.EncodePoint(true)
	if err != nil {
		return errors.New("invalid TargetArbitratorPublicKey")
	}

	strPK := string(publickeyBytes)
	if _, ok := signers[strPK]; ok {
		dns.mux.Unlock()
		log.Warn("arbiter already recorded the schnorr signer")
		return nil
	}
	dns.schnorrWithdrawContentsSigners[hash][strPK] = struct{}{}
	dns.mux.Unlock()

	if len(dns.schnorrWithdrawContentsSigners[hash]) >= getTransactionAgreementArbitratorsCount(
		len(arbitrator.ArbitratorGroupSingleton.GetAllArbitrators())) {
		pks := make([][]byte, 0)
		for k, _ := range dns.schnorrWithdrawContentsSigners[hash] {
			pks = append(pks, []byte(k))
		}

		arbiters := arbitrator.ArbitratorGroupSingleton.GetAllArbitrators()
		pksIndex := make([]uint8, 0)
		for i, a := range arbiters {
			if _, ok := signers[a]; ok {
				pksIndex = append(pksIndex, uint8(i))
			}
		}
		txn.Payload = &payload.WithdrawFromSideChain{
			Signers: pksIndex,
		}

		if err := dns.BroadcastSchnorrWithdrawProposal2(&txn, pks); err != nil {
			return errors.New("failed to BroadcastSchnorrWithdrawProposal2, err:" + err.Error())
		}
	}
	return nil
}

func (dns *DistributedNodeServer) receiveSchnorrWithdrawProposal2Feedback(transactionItem DistributedItem) error {
	return nil
}

func (dns *DistributedNodeServer) receiveSchnorrWithdrawProposal3Feedback(transactionItem DistributedItem) error {
	return nil
}
