package sidechain

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"strconv"

	"encoding/json"
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
	"sync"
)

type SideChainImpl struct {
	mux      sync.Mutex
	isOnDuty bool

	Key           string
	CurrentConfig *config.SideNodeConfig
}

func (client *SideChainImpl) OnP2PReceived(peer *net.Peer, msg p2p.Message) error {
	if msg.CMD() != cs.DepositTxCacheClearCommand {
		return nil
	}

	depositTxClearMsg, ok := msg.(*cs.TxCacheClearMessage)
	if !ok {
		return errors.New("Unknown deposit transaction cache clear message.")
	}

	return store.DbCache.RemoveMainChainTxs(depositTxClearMsg.RemovedTxs)
}

func (sc *SideChainImpl) GetKey() string {
	return sc.Key
}

func (sc *SideChainImpl) getCurrentConfig() *config.SideNodeConfig {
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
	withdrawInfos, err := sc.ParseUserWithdrawTransactionInfo(txn)
	if err != nil {
		return err
	}

	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	transactions := currentArbitrator.CreateWithdrawTransaction(withdrawInfos, sc, txinfo.Hash, &store.DbMainChainFunc{})
	currentArbitrator.BroadcastWithdrawProposal(transactions)

	return nil
}

func (sc *SideChainImpl) OnDutyArbitratorChanged(onDuty bool) {
	sc.mux.Lock()
	sc.isOnDuty = onDuty
	sc.mux.Unlock()
	if onDuty {
		sc.syncSideChainCachedTxs()
		sc.startSidechainMining()
	}
}

func (sc *SideChainImpl) CreateDepositTransaction(target string, proof bloom.MerkleProof, amount common.Fixed64) (*TransactionInfo, error) {
	var totalOutputAmount = amount // The total amount will be spend
	var txOutputs []OutputInfo     // The outputs in transaction

	assetID := spvWallet.SystemAssetId
	txOutput := OutputInfo{
		AssetID:    assetID.String(),
		Value:      totalOutputAmount.String(),
		Address:    target,
		OutputLock: uint32(0),
	}
	txOutputs = append(txOutputs, txOutput)

	spvInfo := new(bytes.Buffer)
	err := proof.Serialize(spvInfo)
	if err != nil {
		return nil, err
	}

	// Create payload
	txPayloadInfo := new(IssueTokenInfo)
	txPayloadInfo.Proof = common.BytesToHexString(spvInfo.Bytes())

	// Create attributes
	txAttr := AttributeInfo{core.Nonce, strconv.FormatInt(rand.Int63(), 10)}
	attributesInfo := make([]AttributeInfo, 0)
	attributesInfo = append(attributesInfo, txAttr)

	// Create program
	program := ProgramInfo{}
	return &TransactionInfo{
		TxType:     core.IssueToken,
		Payload:    txPayloadInfo,
		Attributes: attributesInfo,
		Inputs:     []InputInfo{},
		Outputs:    txOutputs,
		Programs:   []ProgramInfo{program},
		LockTime:   uint32(0),
	}, nil
}

func (sc *SideChainImpl) ParseUserWithdrawTransactionInfo(txn *core.Transaction) ([]*WithdrawInfo, error) {

	var result []*WithdrawInfo

	switch payloadObj := txn.Payload.(type) {
	case *core.PayloadTransferCrossChainAsset:
		for address, index := range payloadObj.AddressesMap {
			info := &WithdrawInfo{
				TargetAddress: address,
				Amount:        txn.Outputs[index].Value,
			}
			result = append(result, info)
		}
	default:
		return nil, errors.New("Invalid payload")
	}

	return result, nil
}

func (sc *SideChainImpl) startSidechainMining() {
	rpcConfig, ok := config.GetRpcConfig(sc.GetKey())
	if ok {
		sideauxpow.StartSidechainMining(rpcConfig)
	}
}

func (sc *SideChainImpl) syncSideChainCachedTxs() {
	txs, err := store.DbCache.GetAllSideChainTxHashes(sc.GetKey())
	if err != nil {
		log.Warn(err)
		return
	}

	//todo update transactions from rpc
	receivedTxs := txs

	if err != nil {
		store.DbCache.RemoveSideChainTxs(receivedTxs)
	}

	cs.P2PClientSingleton.Broadcast(&cs.TxCacheClearMessage{
		Command:    cs.WithdrawTxCacheClearCommand,
		RemovedTxs: receivedTxs})
}
