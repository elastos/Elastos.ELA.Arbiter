package sidechain

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"strconv"
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
	mux         sync.Mutex
	withdrawMux sync.Mutex

	Key           string
	CurrentConfig *config.SideNodeConfig

	LastUsedUtxoHeight        uint32
	LastUsedOutPoints         []core.OutPoint
	ToSendTransactions        map[uint32][]*core.Transaction
	ToSendTransactionsHeight  uint32
	Ready                     bool
	ReceivedUsedUtxoMsgNumber uint32
}

func (client *SideChainImpl) OnP2PReceived(peer *net.Peer, msg p2p.Message) error {
	if msg.CMD() != cs.GetLastArbiterUsedUtxoCommand && msg.CMD() != cs.SendLastArbiterUsedUtxoCommand {
		return nil
	}

	switch m := msg.(type) {
	case *cs.GetLastArbiterUsedUTXOMessage:
		return client.ReceiveGetLastArbiterUsedUtxos(m.Height, m.GenesisAddress)
	case *cs.SendLastArbiterUsedUTXOMessage:
		return client.ReceiveSendLastArbiterUsedUtxos(m.Height, m.GenesisAddress, m.OutPoints)
	}

	return nil
}

func (sc *SideChainImpl) ReceiveSendLastArbiterUsedUtxos(height uint32, genesisAddress string, outPoints []core.OutPoint) error {
	sc.withdrawMux.Lock()
	defer sc.withdrawMux.Unlock()

	sc.mux.Lock()
	log.Info("[ReceiveSendLastArbiterUsedUtxos] Received mssage, received height:", height, "my height:", sc.LastUsedUtxoHeight)
	if sc.GetKey() == genesisAddress && sc.ToSendTransactionsHeight <= height {
		sc.ReceivedUsedUtxoMsgNumber++
		sc.mux.Unlock()
		sc.AddLastUsedOutPoints(outPoints)
		sc.SetLastUsedUtxoHeight(height)
		if sc.Ready && sc.ReceivedUsedUtxoMsgNumber >= config.Parameters.MinReceivedUsedUtxoMsgNumber {
			for _, v := range sc.ToSendTransactions {
				err := sc.CreateAndBroadcastWithdrawProposal(v)
				if err != nil {
					log.Error("[ReceiveSendLastArbiterUsedUtxos] CreateAndBroadcastWithdrawProposal failed")
				}
			}
			sc.mux.Lock()
			sc.Ready = false
			sc.ToSendTransactions = make(map[uint32][]*core.Transaction, 0)
			sc.ToSendTransactionsHeight = 0
			sc.mux.Unlock()
			log.Info("[ReceiveSendLastArbiterUsedUtxos] Send transactions for multi sign")
		}
	}
	return nil
}

func (sc *SideChainImpl) ReceiveGetLastArbiterUsedUtxos(height uint32, genesisAddress string) error {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	if sc.GetKey() == genesisAddress {
		log.Info("[ReceiveGetLastArbiterUsedUtxos] Received mssage, need height:", height, "my height:", sc.LastUsedUtxoHeight)
		if sc.LastUsedUtxoHeight >= height {
			var number = make([]byte, 8)
			var nonce int64
			rand.Read(number)
			binary.Read(bytes.NewReader(number), binary.LittleEndian, &nonce)

			msg := &cs.SendLastArbiterUsedUTXOMessage{
				Command:        cs.SendLastArbiterUsedUtxoCommand,
				GenesisAddress: genesisAddress,
				Height:         sc.LastUsedUtxoHeight,
				OutPoints:      sc.LastUsedOutPoints,
				Nonce:          strconv.FormatInt(nonce, 10),
			}
			msgHash := cs.P2PClientSingleton.GetMessageHash(msg)
			cs.P2PClientSingleton.AddMessageHash(msgHash)
			cs.P2PClientSingleton.Broadcast(msg)

			utxos, err := store.DbCache.UTXOStore.GetAddressUTXOsFromGenesisBlockAddress(genesisAddress)
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

func (sc *SideChainImpl) GetExchangeRate() float32 {
	return sc.getCurrentConfig().Rate
}

func (sc *SideChainImpl) GetCurrentHeight() (uint32, error) {
	return rpc.GetCurrentHeight(sc.getCurrentConfig().Rpc)
}

func (sc *SideChainImpl) GetBlockByHeight(height uint32) (*BlockInfo, error) {
	return rpc.GetBlockByHeight(height, sc.getCurrentConfig().Rpc)
}

func (sc *SideChainImpl) SendTransaction(info *TransactionInfo) (rpc.Response, error) {
	infoBytes, err := json.Marshal(info)
	if err != nil {
		return rpc.Response{}, err
	}

	log.Info("[Rpc-sendtransactioninfo] Deposit transaction to side chain：", sc.CurrentConfig.Rpc.IpAddress, ":", sc.CurrentConfig.Rpc.HttpJsonPort)
	response, err := rpc.CallAndUnmarshalResponse("sendtransactioninfo",
		rpc.Param("Info", common.BytesToHexString(infoBytes)), sc.CurrentConfig.Rpc)
	if err != nil {
		return rpc.Response{}, err
	}

	log.Info("response:", response)
	return response, nil
}

func (sc *SideChainImpl) GetAccountAddress() string {
	return sc.GetKey()
}

func (sc *SideChainImpl) OnUTXOChanged(txinfos []*TransactionInfo, blockHeight uint32) error {
	if len(txinfos) == 0 {
		return errors.New("OnUTXOChanged received txinfos, but size is 0")
	}

	var txHashes []string
	var genesises []string
	var txsBytes [][]byte
	var blockHeights []uint32
	for _, txinfo := range txinfos {
		txn, err := txinfo.ToTransaction()
		if err != nil {
			return err
		}
		txHashes = append(txHashes, txn.Hash().String())
		genesises = append(genesises, sc.GetKey())
		// Serialize transaction
		buf := new(bytes.Buffer)
		txn.Serialize(buf)
		txBytes := buf.Bytes()
		txsBytes = append(txsBytes, txBytes)
		blockHeights = append(blockHeights, blockHeight)
	}

	if err := store.DbCache.SideChainStore.AddSideChainTxs(txHashes, genesises, txsBytes, blockHeights); err != nil {
		return err
	}

	log.Info("[OnUTXOChanged] Find ", len(txHashes), "withdraw transaction, add into dbcache")
	return nil
}

func (sc *SideChainImpl) StartSideChainMining() {
	sideauxpow.StartSideChainMining(sc.CurrentConfig)
}

func (sc *SideChainImpl) SubmitAuxpow(genesishash string, blockhash string, submitauxpow string) error {
	return sideauxpow.SubmitAuxpow(genesishash, blockhash, submitauxpow)
}

func (sc *SideChainImpl) GetExistDepositTransactions(txs []string) ([]string, error) {
	receivedTxs, err := rpc.GetExistDepositTransactions(txs, sc.CurrentConfig.Rpc)
	if err != nil {
		return nil, err
	}
	return receivedTxs, nil
}

func (sc *SideChainImpl) GetTransactionByHash(txHash string) (*core.Transaction, error) {
	txBytes, err := rpc.GetTransactionByHash(txHash, sc.CurrentConfig.Rpc)
	if err != nil {
		return nil, err
	}

	tx := new(core.Transaction)
	reader := bytes.NewReader(txBytes)
	tx.Deserialize(reader)

	return tx, nil
}

func (sc *SideChainImpl) CreateDepositTransaction(depositInfo *DepositInfo, proof bloom.MerkleProof,
	mainChainTransaction *core.Transaction) (*TransactionInfo, error) {
	var txOutputs []OutputInfo // The outputs in transaction

	assetID := spvWallet.SystemAssetId
	rate := common.Fixed64(sc.GetExchangeRate() * 100000000)
	for i := 0; i < len(depositInfo.TargetAddress); i++ {
		amount := depositInfo.CrossChainAmount[i] * rate / 100000000
		txOutput := OutputInfo{
			AssetID:    common.BytesToHexString(common.BytesReverse(assetID.Bytes())),
			Value:      amount.String(),
			Address:    depositInfo.TargetAddress[i],
			OutputLock: uint32(0),
		}
		txOutputs = append(txOutputs, txOutput)
	}

	spvInfo := new(bytes.Buffer)
	err := proof.Serialize(spvInfo)
	if err != nil {
		return nil, err
	}

	transactionBuf := new(bytes.Buffer)
	err = mainChainTransaction.Serialize(transactionBuf)
	if err != nil {
		return nil, err
	}

	// Create payload
	txPayloadInfo := new(RechargeToSideChainInfo)
	txPayloadInfo.Proof = common.BytesToHexString(spvInfo.Bytes())
	txPayloadInfo.MainChainTransaction = common.BytesToHexString(transactionBuf.Bytes())

	// Create attributes
	var number = make([]byte, 8)
	var nonce int64
	rand.Read(number)
	binary.Read(bytes.NewReader(number), binary.LittleEndian, &nonce)

	txAttr := AttributeInfo{core.Nonce, strconv.FormatInt(nonce, 10)}
	attributesInfo := make([]AttributeInfo, 0)
	attributesInfo = append(attributesInfo, txAttr)

	// Create program
	return &TransactionInfo{
		TxType:     core.RechargeToSideChain,
		Payload:    txPayloadInfo,
		Attributes: attributesInfo,
		Inputs:     []InputInfo{},
		Outputs:    txOutputs,
		LockTime:   uint32(0),
	}, nil
}

func (sc *SideChainImpl) ParseUserWithdrawTransactionInfos(txn []*core.Transaction) (*WithdrawInfo, error) {
	result := new(WithdrawInfo)
	for _, tx := range txn {
		payloadObj, ok := tx.Payload.(*core.PayloadTransferCrossChainAsset)
		if !ok {
			return nil, errors.New("Invalid payload")
		}
		for i := 0; i < len(payloadObj.CrossChainAddress); i++ {
			result.TargetAddress = append(result.TargetAddress, payloadObj.CrossChainAddress[i])
			result.Amount = append(result.Amount, tx.Outputs[payloadObj.OutputIndex[i]].Value)
			result.CrossChainAmount = append(result.CrossChainAmount, payloadObj.CrossChainAmount[i])
		}
	}
	return result, nil
}

func (sc *SideChainImpl) SendCachedWithdrawTxs() error {
	txHashes, blockHeights, err := store.DbCache.SideChainStore.GetAllSideChainTxHashesAndHeights(sc.GetKey())
	if err != nil {
		return err
	}
	receivedTxs, err := rpc.GetExistWithdrawTransactions(txHashes)
	if err != nil {
		return err
	}

	unsolvedTxs, unsolvedBlockHeights := SubstractTransactionHashesAndBlockHeights(txHashes, blockHeights, receivedTxs)
	unsolvedTransactions, err := store.DbCache.SideChainStore.GetSideChainTxsFromHashes(unsolvedTxs)
	if err != nil {
		return err
	}

	if len(unsolvedTransactions) == 0 {
		return nil
	}

	chainHeight, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return err
	}

	heightTxsMap := GetHeightTransactionsMap(unsolvedTransactions, unsolvedBlockHeights)

	sc.ToSendTransactions = heightTxsMap
	sc.ToSendTransactionsHeight = chainHeight - 1
	sc.Ready = true
	sc.ReceivedUsedUtxoMsgNumber = 0

	var number = make([]byte, 8)
	var nonce int64
	rand.Read(number)
	binary.Read(bytes.NewReader(number), binary.LittleEndian, &nonce)

	msg := &cs.GetLastArbiterUsedUTXOMessage{
		Command:        cs.GetLastArbiterUsedUtxoCommand,
		GenesisAddress: sc.GetKey(),
		Height:         chainHeight - 1,
		Nonce:          strconv.FormatInt(nonce, 10)}
	msgHash := cs.P2PClientSingleton.GetMessageHash(msg)
	cs.P2PClientSingleton.AddMessageHash(msgHash)
	cs.P2PClientSingleton.Broadcast(msg)
	log.Info("[SendCachedWithdrawTxs] Find withdraw transaction, send GetLastArbiterUsedUtxoCommand mssage")

	err = store.DbCache.SideChainStore.RemoveSideChainTxs(receivedTxs)
	if err != nil {
		return err
	}

	err = store.FinishedTxsDbCache.AddSucceedWIthdrawTx(receivedTxs)
	if err != nil {
		return err
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

	log.Info("[CreateAndBroadcastWithdrawProposal] Transactions count: ", len(transactions))
	currentArbitrator.BroadcastWithdrawProposal(transactions)

	return nil
}
