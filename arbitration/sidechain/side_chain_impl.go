package sidechain

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"strconv"

	"encoding/json"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA.SPV/net"
	spvWallet "github.com/elastos/Elastos.ELA.SPV/spvwallet"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
	"github.com/elastos/Elastos.ELA/bloom"
	"github.com/elastos/Elastos.ELA/core"
)

type SideChainImpl struct {
	mux      sync.Mutex
	isOnDuty bool

	Key           string
	CurrentConfig *config.SideNodeConfig

	tick int

	LastUsedUtxoHeight        uint32
	LastUsedOutPoints         []core.OutPoint
	ToSendTransactions        []*core.Transaction
	Ready                     bool
	ReceivedUsedUtxoMsgNumber uint32
}

func (client *SideChainImpl) OnP2PReceived(peer *net.Peer, msg p2p.Message) error {
	if msg.CMD() != cs.DepositTxCacheClearCommand && msg.CMD() != cs.GetLastArbiterUsedUtxoCommand &&
		msg.CMD() != cs.SendLastArbiterUsedUtxoCommand {
		return nil
	}

	switch m := msg.(type) {
	case *cs.TxCacheClearMessage:
		return store.DbCache.RemoveMainChainTxs(m.RemovedTxs)
	case *cs.GetLastArbiterUsedUTXOMessage:
		return client.ReceiveGetLastArbiterUsedUtxos(m.Height, m.GenesisAddress)
	case *cs.SendLastArbiterUsedUTXOMessage:
		return client.ReceiveSendLastArbiterUsedUtxos(m.Height, m.GenesisAddress, m.OutPoints)
	}

	return nil
}

func (sc *SideChainImpl) ReceiveSendLastArbiterUsedUtxos(height uint32, genesisAddress string, outPoints []core.OutPoint) error {
	sc.mux.Lock()
	log.Info("ReceiveSendLastArbiterUsedUtxos receive mssage, received height:", height, "my height:", sc.LastUsedUtxoHeight)
	if sc.GetKey() == genesisAddress && sc.LastUsedUtxoHeight <= height {
		sc.ReceivedUsedUtxoMsgNumber++
		sc.mux.Unlock()
		sc.AddLastUsedOutPoints(outPoints)
		sc.SetLastUsedUtxoHeight(height)
		if !sc.Ready && sc.ReceivedUsedUtxoMsgNumber >= config.Parameters.MinReceivedUsedUtxoMsgNumber {
			err := sc.CreateAndBroadcastWithdrawProposal(sc.ToSendTransactions)
			if err != nil {
				log.Error("CreateAndBroadcastWithdrawProposal failed")
			}
			sc.mux.Lock()
			sc.Ready = true
			sc.ToSendTransactions = make([]*core.Transaction, 0)
			sc.mux.Unlock()
			log.Info("ReceiveSendLastArbiterUsedUtxos CreateAndBroadcastWithdrawProposal transactions")
		}
	}
	return nil
}

func (sc *SideChainImpl) ReceiveGetLastArbiterUsedUtxos(height uint32, genesisAddress string) error {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	if sc.GetKey() == genesisAddress {
		log.Info("ReceiveGetLastArbiterUsedUtxos receive mssage, need height:", height, "my height:", sc.LastUsedUtxoHeight)
		if sc.LastUsedUtxoHeight >= height {
			msg := &cs.SendLastArbiterUsedUTXOMessage{
				Command:        cs.SendLastArbiterUsedUtxoCommand,
				GenesisAddress: genesisAddress,
				Height:         sc.LastUsedUtxoHeight,
				OutPoints:      sc.LastUsedOutPoints}
			msgHash := cs.P2PClientSingleton.GetMessageHash(msg)
			cs.P2PClientSingleton.AddMessageHash(msgHash)
			cs.P2PClientSingleton.Broadcast(msg)

			utxos, err := store.DbCache.GetAddressUTXOsFromGenesisBlockAddress(genesisAddress)
			if err != nil {
				return err
			}
			var newOutPoints []core.OutPoint
			for _, op := range sc.LastUsedOutPoints {
				isContained := false
				for _, utxo := range utxos {
					if op.IsEqual(utxo.Input.Previous) {
						isContained = true
					}
				}
				if !isContained {
					newOutPoints = append(newOutPoints, op)
				}
			}
			sc.LastUsedOutPoints = newOutPoints
		} else {
			return errors.New("I have no needed outpoints at requested height")
		}
	}

	return nil
}

func (sc *SideChainImpl) GetKey() string {
	return sc.Key
}

func (sc *SideChainImpl) GetLastUsedUtxoHeight() uint32 {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	return sc.LastUsedUtxoHeight
}

func (sc *SideChainImpl) SetLastUsedUtxoHeight(height uint32) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	sc.LastUsedUtxoHeight = height
}

func (sc *SideChainImpl) GetLastUsedOutPoints() []core.OutPoint {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	return sc.LastUsedOutPoints
}

func (sc *SideChainImpl) AddLastUsedOutPoints(ops []core.OutPoint) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	for _, op := range ops {
		isContained := false
		for _, outPoint := range sc.LastUsedOutPoints {
			if op.IsEqual(outPoint) {
				isContained = true
			}
		}
		if !isContained {
			sc.LastUsedOutPoints = append(sc.LastUsedOutPoints, op)
		}
	}
}

func (sc *SideChainImpl) RemoveLastUsedOutPoints(ops []core.OutPoint) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	var newOutPoints []core.OutPoint
	for _, outPoint := range sc.LastUsedOutPoints {
		isContained := false
		for _, op := range ops {
			if outPoint.IsEqual(op) {
				isContained = true
			}
		}
		if !isContained {
			newOutPoints = append(newOutPoints, outPoint)
		}
	}
	sc.LastUsedOutPoints = newOutPoints
}

func (sc *SideChainImpl) getCurrentConfig() *config.SideNodeConfig {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	if sc.CurrentConfig == nil {
		for _, sideConfig := range config.Parameters.SideNodeList {
			if sc.GetKey() == sideConfig.GenesisBlockAddress {
				sc.CurrentConfig = sideConfig
				break
			}
		}
	}
	return sc.CurrentConfig
}

func (sc *SideChainImpl) IsOnDuty() bool {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	return sc.isOnDuty
}

func (sc *SideChainImpl) GetRage() float32 {
	return sc.getCurrentConfig().Rate
}

func (sc *SideChainImpl) GetTick() int {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	return sc.tick
}

func (sc *SideChainImpl) SetTick(tick int) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	sc.tick = tick
}

func (sc *SideChainImpl) GetCurrentHeight() (uint32, error) {
	return rpc.GetCurrentHeight(sc.getCurrentConfig().Rpc)
}

func (sc *SideChainImpl) GetBlockByHeight(height uint32) (*BlockInfo, error) {
	return rpc.GetBlockByHeight(height, sc.getCurrentConfig().Rpc)
}

func (sc *SideChainImpl) SendTransaction(info *TransactionInfo) error {
	infoBytes, err := json.Marshal(info)
	if err != nil {
		return err
	}

	result, err := rpc.CallAndUnmarshal("sendtransactioninfo",
		rpc.Param("Info", common.BytesToHexString(infoBytes)), sc.CurrentConfig.Rpc)
	if err != nil {
		return err
	}

	fmt.Println(result)
	return nil
}

func (sc *SideChainImpl) GetAccountAddress() string {
	return sc.GetKey()
}

func (sc *SideChainImpl) OnUTXOChanged(txinfo *TransactionInfo) error {

	txn, err := txinfo.ToTransaction()
	if err != nil {
		return err
	}

	if err := store.DbCache.AddSideChainTx(txn.Hash().String(),
		sc.GetKey(), txn); err != nil {
		return err
	}

	log.Info("OnUTXOChanged find withdraw transaction, add into dbcache")

	/*if !sc.IsOnDuty() { //only on duty arbitrator need to broadcast withdraw proposal
		return nil
	}

	return sc.createAndBroadcastWithdrawProposal(txn)*/
	return nil
}

func (sc *SideChainImpl) OnDutyArbitratorChanged(onDuty bool) {
	sc.mux.Lock()
	sc.isOnDuty = onDuty
	sc.mux.Unlock()
	if onDuty {
		err := sc.syncSideChainCachedTxs()
		if err != nil {
			log.Warn(err)
		}
	}
}

func (sc *SideChainImpl) StartSidechainMining() {
	sideauxpow.StartSidechainMining(sc.CurrentConfig)
}

func (sc *SideChainImpl) GetExistDepositTransactions(txs []string) ([]string, error) {
	receivedTxs, err := rpc.GetExistDepositTransactions(txs, sc.CurrentConfig.Rpc)
	if err != nil {
		return nil, err
	}
	return receivedTxs, nil
}

func (sc *SideChainImpl) CreateDepositTransaction(infoArray []*DepositInfo, proof bloom.MerkleProof,
	mainChainTransaction *core.Transaction) (*TransactionInfo, error) {
	var txOutputs []OutputInfo // The outputs in transaction

	assetID := spvWallet.SystemAssetId
	rateFloat := sc.GetRage()
	for _, info := range infoArray {
		amount := info.CrossChainAmount * common.Fixed64(rateFloat)
		txOutput := OutputInfo{
			AssetID:    assetID.String(),
			Value:      amount.String(),
			Address:    info.TargetAddress,
			OutputLock: uint32(0),
		}
		txOutputs = append(txOutputs, txOutput)
	}

	spvInfo := new(bytes.Buffer)
	err := proof.Serialize(spvInfo)
	if err != nil {
		return nil, err
	}

	transactionInfo := new(bytes.Buffer)
	err = mainChainTransaction.Serialize(transactionInfo)
	if err != nil {
		return nil, err
	}

	// Create payload
	txPayloadInfo := new(IssueTokenInfo)
	txPayloadInfo.Proof = common.BytesToHexString(spvInfo.Bytes())
	txPayloadInfo.MainChainTransaction = common.BytesToHexString(transactionInfo.Bytes())

	// Create attributes
	txAttr := AttributeInfo{core.Nonce, strconv.FormatInt(rand.Int63(), 10)}
	attributesInfo := make([]AttributeInfo, 0)
	attributesInfo = append(attributesInfo, txAttr)

	// Create program
	return &TransactionInfo{
		TxType:     core.IssueToken,
		Payload:    txPayloadInfo,
		Attributes: attributesInfo,
		Inputs:     []InputInfo{},
		Outputs:    txOutputs,
		LockTime:   uint32(0),
	}, nil
}

func (sc *SideChainImpl) ParseUserWithdrawTransactionInfo(txn *core.Transaction) ([]*WithdrawInfo, error) {
	var result []*WithdrawInfo

	switch payloadObj := txn.Payload.(type) {
	case *core.PayloadTransferCrossChainAsset:
		for i := 0; i < len(payloadObj.CrossChainAddress); i++ {
			info := &WithdrawInfo{
				TargetAddress:    payloadObj.CrossChainAddress[i],
				Amount:           txn.Outputs[payloadObj.OutputIndex[i]].Value,
				CrossChainAmount: payloadObj.CrossChainAmount[i],
			}
			result = append(result, info)
		}
	default:
		return nil, errors.New("Invalid payload")
	}

	return result, nil
}

func (sc *SideChainImpl) ParseUserWithdrawTransactionInfos(txn []*core.Transaction) ([]*WithdrawInfo, error) {
	var result []*WithdrawInfo

	for _, tx := range txn {
		switch payloadObj := tx.Payload.(type) {
		case *core.PayloadTransferCrossChainAsset:
			for i := 0; i < len(payloadObj.CrossChainAddress); i++ {
				info := &WithdrawInfo{
					TargetAddress:    payloadObj.CrossChainAddress[i],
					Amount:           tx.Outputs[payloadObj.OutputIndex[i]].Value,
					CrossChainAmount: payloadObj.CrossChainAmount[i],
				}
				result = append(result, info)
			}
		default:
			return nil, errors.New("Invalid payload")
		}
	}

	return result, nil
}

func (sc *SideChainImpl) syncSideChainCachedTxs() error {
	txs, err := store.DbCache.GetAllSideChainTxHashes(sc.GetKey())
	if err != nil {
		return err
	}

	receivedTxs, err := rpc.GetExistWithdrawTransactions(txs)
	if err != nil {
		return err
	}

	unsolvedTxs := SubstractTransactionHashes(txs, receivedTxs)
	transactions, err := store.DbCache.GetSideChainTxsFromHashes(unsolvedTxs)
	if err != nil {
		return err
	}

	if len(transactions) == 0 {
		log.Info("No withdraw transaction to deal with")
		return nil
	}

	chainHeight, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return err
	}

	/*if sc.LastUsedUtxoHeight == chainHeight-1 || sc.LastUsedUtxoHeight == 0 {
		err := sc.CreateAndBroadcastWithdrawProposal(transactions)
		if err != nil {
			return err
		}
		sc.LastUsedUtxoHeight = chainHeight
	} else*/{
		sc.ToSendTransactions = transactions
		sc.Ready = false
		sc.ReceivedUsedUtxoMsgNumber = 0
		msg := &cs.GetLastArbiterUsedUTXOMessage{
			Command:        cs.GetLastArbiterUsedUtxoCommand,
			GenesisAddress: sc.GetKey(),
			Height:         chainHeight - 1,
			Nonce:          strconv.FormatInt(rand.Int63(), 10)}
		msgHash := cs.P2PClientSingleton.GetMessageHash(msg)
		cs.P2PClientSingleton.AddMessageHash(msgHash)
		cs.P2PClientSingleton.Broadcast(msg)
		log.Info("syncSideChainCachedTxs find withdraw transaction, send GetLastArbiterUsedUtxoCommand mssage")
	}

	err = store.DbCache.RemoveSideChainTxs(receivedTxs)
	if err != nil {
		return err
	}

	if len(receivedTxs) != 0 {
		msg := &cs.TxCacheClearMessage{
			Command:    cs.WithdrawTxCacheClearCommand,
			RemovedTxs: receivedTxs}
		cs.P2PClientSingleton.AddMessageHash(cs.P2PClientSingleton.GetMessageHash(msg))
		cs.P2PClientSingleton.Broadcast(msg)
	}

	return nil
}

func (sc *SideChainImpl) CreateAndBroadcastWithdrawProposal(txns []*core.Transaction) error {
	withdrawInfos, err := sc.ParseUserWithdrawTransactionInfos(txns)
	if err != nil {
		return err
	}

	var txHashes []string
	for _, txn := range txns {
		txHashes = append(txHashes, txn.Hash().String())
	}

	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	transactions := currentArbitrator.CreateWithdrawTransactions(withdrawInfos, sc, txHashes, &store.DbMainChainFunc{})
	currentArbitrator.BroadcastWithdrawProposal(transactions)

	return nil
}
