package cs

import (
	"bytes"
	"encoding/hex"
	"errors"
	"sort"
	"time"

	"math/big"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	crypto2 "github.com/elastos/Elastos.ELA.Arbiter/arbitration/crypto"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
)

const (
	MCErrDoubleSpend          int64 = 45010
	MCErrSidechainTxDuplicate int64 = 45012

	// time: wait schnorr signers feedback
	SchnorrFeedbackInterval time.Duration = time.Second * 5
)

type DistributedNodeServer struct {
	mux                       *sync.Mutex
	withdrawMux               *sync.Mutex
	unsolvedContents          map[common.Uint256]base.DistributedContent
	unsolvedContentsSignature map[common.Uint256]map[common.Uint160]struct{}

	// schnorr withdraw
	schnorrWithdrawContentsTransaction     map[common.Uint256]types.Transaction // key: nonce hash
	schnorrWithdrawRequestRContentsSigners map[common.Uint256]map[string]KRP
	schnorrWithdrawRequestSContentsSigners map[common.Uint256]map[string]*big.Int

	// no need to reset, just record unsigned count
	UnsignedSigners map[string]uint64
}

func (dns *DistributedNodeServer) Reset() {
	dns.unsolvedContents = make(map[common.Uint256]base.DistributedContent)
	dns.unsolvedContentsSignature = make(map[common.Uint256]map[common.Uint160]struct{})
	dns.schnorrWithdrawContentsTransaction = make(map[common.Uint256]types.Transaction)
	dns.schnorrWithdrawRequestRContentsSigners = make(map[common.Uint256]map[string]KRP)
	dns.schnorrWithdrawRequestSContentsSigners = make(map[common.Uint256]map[string]*big.Int)
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
	if dns.schnorrWithdrawRequestRContentsSigners == nil {
		dns.schnorrWithdrawRequestRContentsSigners = make(map[common.Uint256]map[string]KRP)
	}
	if dns.schnorrWithdrawRequestSContentsSigners == nil {
		dns.schnorrWithdrawRequestSContentsSigners = make(map[common.Uint256]map[string]*big.Int)
	}
	if dns.UnsignedSigners == nil {
		dns.UnsignedSigners = make(map[string]uint64)
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

func CreateSchnonrrRedeemScript(px *big.Int, py *big.Int) ([]byte, error) {
	var sumPublicKey [33]byte
	copy(sumPublicKey[:], crypto2.Marshal(px, py))
	publicKey, err := crypto.DecodePoint(sumPublicKey[:])
	if err != nil {
		return nil, err
	}
	return contract.CreateSchnorrMultiSigRedeemScript(publicKey)
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

func (dns *DistributedNodeServer) sendSchnorrItemMsgToSelf(nonceHash common.Uint256) {
	go func() {
		time.Sleep(SchnorrFeedbackInterval)
		P2PClientSingleton.messageQueue <- &messageItem{
			[33]byte{}, &SendSchnorrProposalMessage{NonceHash: nonceHash}}
	}()
}

func (dns *DistributedNodeServer) recordKRPOfMyself(nonceHash common.Uint256) error {
	// record KRP of myself
	currentAccount := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	strPK := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitratorPublicKey()
	k0, rx, ry, px, py, err := currentAccount.GetSchnorrR()
	if err != nil {
		return err
	}
	dns.schnorrWithdrawRequestRContentsSigners[nonceHash][strPK] = KRP{
		K0: k0,
		Rx: rx,
		Ry: ry,
		Px: px,
		Py: py,
	}
	return nil
}

func (dns *DistributedNodeServer) BroadcastSchnorrWithdrawProposal2(txn *types.Transaction) error {
	var txType TransactionType
	switch txn.TxType {
	case types.WithdrawFromSideChain:
		txType = WithdrawTransaction
	case types.ReturnCRDepositCoin:
		txType = ReturnDepositTransaction
	}

	content := SchnorrWithdrawRequestRProposalContent{
		Nonce: txn.Hash().Bytes()}
	proposal, err := dns.generateDistributedSchnorrProposal2(
		txn, txType, SchnorrMultisigContent2,
		content)
	if err != nil {
		return err
	}

	nonceHash := content.Hash()
	if err := dns.recordKRPOfMyself(nonceHash); err != nil {
		return err
	}
	dns.sendToArbitrator(proposal)
	dns.sendSchnorrItemMsgToSelf(nonceHash)

	return nil
}

func (dns *DistributedNodeServer) BroadcastSchnorrWithdrawProposal3(
	txn *types.Transaction, pks [][]byte, e *big.Int) error {
	var txType TransactionType
	switch txn.TxType {
	case types.WithdrawFromSideChain:
		txType = WithdrawTransaction
	case types.ReturnCRDepositCoin:
		txType = ReturnDepositTransaction
	}

	proposal, err := dns.generateDistributedSchnorrProposal3(
		txn, txType, SchnorrMultisigContent3,
		SchnorrWithdrawRequestSProposalContent{
			Tx:         txn,
			Publickeys: pks,
			E:          e})
	if err != nil {
		return err
	}

	dns.sendToArbitrator(proposal)
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

func (dns *DistributedNodeServer) generateDistributedSchnorrProposal2(
	txn *types.Transaction, txType TransactionType,
	cType DistributeContentType, content SchnorrWithdrawRequestRProposalContent) ([]byte, error) {
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
		TargetArbitratorPublicKey:      currentArbitrator.GetPublicKey(),
		TargetArbitratorProgramHash:    programHash,
		TransactionType:                txType,
		Type:                           cType,
		SchnorrRequestRProposalContent: content,
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
	dns.schnorrWithdrawRequestRContentsSigners[content.Hash()] = make(map[string]KRP)
	return nil, nil
}

func (dns *DistributedNodeServer) generateDistributedSchnorrProposal3(
	txn *types.Transaction, txType TransactionType,
	cType DistributeContentType, content SchnorrWithdrawRequestSProposalContent) ([]byte, error) {
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
		TargetArbitratorPublicKey:      currentArbitrator.GetPublicKey(),
		TargetArbitratorProgramHash:    programHash,
		TransactionType:                txType,
		Type:                           cType,
		SchnorrRequestSProposalContent: content,
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
	dns.schnorrWithdrawRequestSContentsSigners[content.Hash()] = make(map[string]*big.Int)
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
	case AnswerSchnorrMultisigContent2:
		return dns.receiveSchnorrWithdrawProposal2Feedback(transactionItem)
	case AnswerSchnorrMultisigContent3:
		return dns.receiveSchnorrWithdrawProposal3Feedback(transactionItem)
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

func (dns *DistributedNodeServer) receiveSchnorrWithdrawProposal2Feedback(transactionItem DistributedItem) error {
	if err := transactionItem.CheckSchnorrFeedbackRequestRSignedData(); err != nil {
		return err
	}

	dns.mux.Lock()
	if dns.schnorrWithdrawContentsTransaction == nil {
		dns.mux.Unlock()
		return errors.New("can not find proposal")
	}
	hash := transactionItem.SchnorrRequestRProposalContent.Hash()
	_, ok := dns.schnorrWithdrawContentsTransaction[hash]
	if !ok {
		dns.mux.Unlock()
		return errors.New("can not find proposal transaction")
	}
	signers, ok := dns.schnorrWithdrawRequestRContentsSigners[hash]
	if !ok {
		dns.mux.Unlock()
		return errors.New("can not find RequestR signer")
	}
	pkBuf, err := transactionItem.TargetArbitratorPublicKey.EncodePoint(true)
	if err != nil {
		dns.mux.Unlock()
		return errors.New("invalid TargetArbitratorPublicKey")
	}
	strPK := string(pkBuf)
	if _, ok := signers[strPK]; ok {
		dns.mux.Unlock()
		log.Warn("arbiter already recorded the schnorr signer")
		return nil
	}
	dns.schnorrWithdrawRequestRContentsSigners[hash][strPK] = transactionItem.SchnorrRequestRProposalContent.R
	dns.mux.Unlock()

	return nil
}

func (dns *DistributedNodeServer) ReceiveSendSchnorrWithdrawProposal3(nonceHash common.Uint256) error {
	myPK := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitratorPublicKey()

	// try check signers count
	dns.mux.Lock()
	defer dns.mux.Unlock()

	if dns.schnorrWithdrawContentsTransaction == nil {
		return errors.New("can not find proposal")
	}
	txn, ok := dns.schnorrWithdrawContentsTransaction[nonceHash]
	if !ok {
		return errors.New("can not find proposal transaction")
	}
	signers, ok := dns.schnorrWithdrawRequestRContentsSigners[nonceHash]
	if !ok {
		return errors.New("can not find RequestR signer")
	}

	minSignersCount := getTransactionAgreementArbitratorsCount(
		len(arbitrator.ArbitratorGroupSingleton.GetAllArbitrators()))
	if len(dns.schnorrWithdrawRequestRContentsSigners[nonceHash]) >= minSignersCount {
		// random select signers
		sortedSigners := make([]string, len(signers))
		for k, _ := range signers {
			sortedSigners = append(sortedSigners, k)
		}
		sort.Slice(sortedSigners, func(i, j int) bool {
			// myself need to be the first one
			if sortedSigners[i] == myPK {
				return true
			}
			if sortedSigners[j] == myPK {
				return false
			}
			if _, ok := dns.UnsignedSigners[sortedSigners[i]]; !ok {
				return true
			}
			return dns.UnsignedSigners[sortedSigners[i]] < dns.UnsignedSigners[sortedSigners[j]]
		})

		randomSigners := make(map[string]KRP)
		for _, k := range sortedSigners {
			randomSigners[k] = signers[k]
			if len(randomSigners) == minSignersCount {
				break
			}
		}

		arbiters := arbitrator.ArbitratorGroupSingleton.GetAllArbitrators()
		pksIndex := make([]uint8, 0)
		pks := make([][]byte, 0)
		for i, a := range arbiters {
			if _, ok := randomSigners[a]; ok {
				pksIndex = append(pksIndex, uint8(i))
				pks = append(pks, []byte(a))
			}
		}
		txn.Payload = &payload.WithdrawFromSideChain{
			Signers: pksIndex,
		}

		// get pxs pys rxs rys
		var pxs, pys, rxs, rys []*big.Int
		for _, v := range dns.schnorrWithdrawRequestRContentsSigners[nonceHash] {
			pxs = append(pxs, v.Px)
			pys = append(pys, v.Py)
			rxs = append(rxs, v.Rx)
			rys = append(rys, v.Ry)
		}

		// get E
		message := txn.Hash()
		e := crypto2.GetE(rxs, rys, pxs, pys, message[:])

		if err := dns.BroadcastSchnorrWithdrawProposal3(&txn, pks, e); err != nil {
			return errors.New("failed to BroadcastSchnorrWithdrawProposal2, err:" + err.Error())
		}

		// record signature of myself
		currentAccount := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
		dns.schnorrWithdrawRequestSContentsSigners[nonceHash][myPK] = currentAccount.GetSchnorrS(e)
	} else {
		log.Errorf("[ReceiveSendSchnorrWithdrawProposal3] not enought "+
			"signers for transaction %s, need %d, current %d",
			nonceHash, minSignersCount, len(dns.schnorrWithdrawRequestRContentsSigners[nonceHash]))
	}

	// record unsigned signers
	arbiters := arbitrator.ArbitratorGroupSingleton.GetAllArbitrators()
	for _, a := range arbiters {
		if _, ok := signers[a]; !ok {
			if count, ok := dns.UnsignedSigners[a]; ok {
				dns.UnsignedSigners[a] = count + 1
			} else {
				dns.UnsignedSigners[a] = 1
			}
		}

		// record for next proposal, if someone signed it and will mul
		if a == myPK {
			continue
		}
		if count, ok := dns.UnsignedSigners[a]; ok {
			dns.UnsignedSigners[a] = count + 1
		} else {
			dns.UnsignedSigners[a] = 1
		}
	}

	return nil
}

func (dns *DistributedNodeServer) receiveSchnorrWithdrawProposal3Feedback(transactionItem DistributedItem) error {
	if err := transactionItem.CheckSchnorrFeedbackRequestRSignedData(); err != nil {
		return err
	}

	dns.mux.Lock()
	if dns.schnorrWithdrawContentsTransaction == nil {
		dns.mux.Unlock()
		return errors.New("can not find proposal")
	}
	hash := transactionItem.SchnorrRequestRProposalContent.Hash()
	txn, ok := dns.schnorrWithdrawContentsTransaction[hash]
	if !ok {
		dns.mux.Unlock()
		return errors.New("can not find proposal transaction")
	}
	signers, ok := dns.schnorrWithdrawRequestRContentsSigners[hash]
	if !ok {
		dns.mux.Unlock()
		return errors.New("can not find RequestR signer")
	}
	_, ok = dns.schnorrWithdrawRequestSContentsSigners[hash]
	if !ok {
		dns.mux.Unlock()
		return errors.New("can not find RequestS signer")
	}
	publickeyBytes, err := transactionItem.TargetArbitratorPublicKey.EncodePoint(true)
	if err != nil {
		dns.mux.Unlock()
		return errors.New("invalid TargetArbitratorPublicKey")
	}

	strPK := string(publickeyBytes)
	if _, ok := signers[strPK]; ok {
		dns.mux.Unlock()
		log.Warn("arbiter already recorded the schnorr signer")
		return nil
	}

	if count, ok := dns.UnsignedSigners[strPK]; !ok || count < 1 {
		return errors.New("not found in UnsignedSigners")
	} else {
		dns.UnsignedSigners[strPK] = count - 1
	}

	dns.schnorrWithdrawRequestSContentsSigners[hash][strPK] = transactionItem.SchnorrRequestSProposalContent.S

	if len(dns.schnorrWithdrawRequestRContentsSigners[hash]) == len(dns.schnorrWithdrawRequestRContentsSigners[hash]) {
		// aggregate signatures
		Px, Py := new(big.Int), new(big.Int)
		Rx, Ry := new(big.Int), new(big.Int)
		for _, v := range dns.schnorrWithdrawRequestRContentsSigners[hash] {
			Rx, Ry = crypto2.Curve.Add(Rx, Ry, v.Rx, v.Ry)
			Px, Py = crypto2.Curve.Add(Px, Py, v.Px, v.Py)
		}

		s := new(big.Int).SetInt64(0)
		for pk, v := range dns.schnorrWithdrawRequestRContentsSigners[hash] {
			if _, ok := dns.schnorrWithdrawRequestSContentsSigners[hash][pk]; !ok {
				return errors.New("invalid schnorrWithdrawRequestSContentsSigners, not found signature of " + pk)
			}
			k := crypto2.GetK(Ry, v.K0)
			k.Add(k, dns.schnorrWithdrawRequestSContentsSigners[hash][pk])
			s.Add(s, k)
		}
		dns.mux.Unlock()

		signature := crypto2.GetS(Rx, s)

		// create transaction with schnorr signature
		redeemScript, err := CreateSchnonrrRedeemScript(Px, Py)
		if err != nil {
			return errors.New("failed to CreateSchnonrrRedeemScript, " + err.Error())
		}
		p := &program.Program{
			Code:      redeemScript,
			Parameter: signature[:],
		}
		txn.Programs = []*program.Program{p}

		// broadcast the schnorr transaction to main chain
		c := TxDistributedContent{
			Tx: &txn,
		}
		if err = c.Submit(); err != nil {
			log.Warn(err.Error())
			return err
		}
	} else {
		dns.mux.Unlock()
	}

	return nil
}
