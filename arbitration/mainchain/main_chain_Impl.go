package mainchain

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	. "github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA.SPV/net"
	spvWallet "github.com/elastos/Elastos.ELA.SPV/spvwallet"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
	. "github.com/elastos/Elastos.ELA/core"
)

const WithdrawAssetLockTime uint32 = 6

type MainChainImpl struct {
	*DistributedNodeServer
}

func (dns *MainChainImpl) SyncMainChainCachedTxs() error {
	txs, err := DbCache.GetAllMainChainTxs()
	if err != nil {
		return err
	}

	//todo sync from rpc
	receivedTxs := txs

	err = DbCache.RemoveMainChainTxs(receivedTxs)
	if err != nil {
		log.Warn(err)
	}

	msg := &TxCacheClearMessage{Command: DepositTxCacheClearCommand, RemovedTxs: receivedTxs}
	P2PClientSingleton.Broadcast(msg)
	return nil
}

func (dns *MainChainImpl) OnP2PReceived(peer *net.Peer, msg p2p.Message) error {
	if msg.CMD() != dns.P2pCommand || msg.CMD() != WithdrawTxCacheClearCommand {
		return nil
	}

	switch m := msg.(type) {
	case *SignMessage:
		return dns.ReceiveProposalFeedback(m.Content)
	case *TxCacheClearMessage:
		return DbCache.RemoveSideChainTxs(m.RemovedTxs)
	}
	return nil
}

func (mc *MainChainImpl) CreateWithdrawTransaction(withdrawBank string, target string, amount Fixed64,
	sideChainTransactionHash string, mcFunc MainChainFunc) (*Transaction, error) {

	mc.syncChainData()

	// Check if from address is valid
	assetID := spvWallet.SystemAssetId
	programhash, err := Uint168FromAddress(target)
	if err != nil {
		return nil, err
	}
	// Create transaction outputs
	var txOutputs []*Output
	txOutput := &Output{
		AssetID:     Uint256(assetID),
		ProgramHash: *programhash,
		Value:       amount,
		OutputLock:  uint32(WithdrawAssetLockTime),
	}

	txOutputs = append(txOutputs, txOutput)

	var txInputs []*Input
	availableUTXOs, err := mcFunc.GetAvailableUtxos(withdrawBank)
	if err != nil {
		return nil, err
	}

	// Create transaction inputs
	var totalOutputAmount = amount
	for _, utxo := range availableUTXOs {
		txInputs = append(txInputs, utxo.Input)
		if *utxo.Amount < totalOutputAmount {
			totalOutputAmount -= *utxo.Amount
		} else if *utxo.Amount == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if *utxo.Amount > totalOutputAmount {
			programHash, err := Uint168FromAddress(withdrawBank)
			if err != nil {
				return nil, err
			}
			change := &Output{
				AssetID:     Uint256(spvWallet.SystemAssetId),
				Value:       Fixed64(*utxo.Amount - totalOutputAmount),
				OutputLock:  uint32(0),
				ProgramHash: *programHash,
			}
			txOutputs = append(txOutputs, change)
			totalOutputAmount = 0
			break
		}
	}

	if totalOutputAmount > 0 {
		return nil, errors.New("Available token is not enough")
	}
	if err != nil {
		return nil, err
	}

	redeemScript, err := CreateRedeemScript()
	if err != nil {
		return nil, err
	}

	// Create payload
	chainHeight, err := mcFunc.GetMainNodeCurrentHeight()
	if err != nil {
		return nil, err
	}

	txPayload := &PayloadWithdrawAsset{
		BlockHeight:              chainHeight,
		GenesisBlockAddress:      withdrawBank,
		SideChainTransactionHash: sideChainTransactionHash}
	program := &Program{redeemScript, nil}

	// Create attributes
	txAttr := NewAttribute(Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	attributes := make([]*Attribute, 0)
	attributes = append(attributes, &txAttr)

	return &Transaction{
		TxType:     WithdrawAsset,
		Payload:    txPayload,
		Attributes: attributes,
		Inputs:     txInputs,
		Outputs:    txOutputs,
		Programs:   []*Program{program},
		LockTime:   uint32(0),
	}, nil
}

func (mc *MainChainImpl) ParseUserDepositTransactionInfo(txn *Transaction) ([]*DepositInfo, error) {

	var result []*DepositInfo

	switch payloadObj := txn.Payload.(type) {
	case *PayloadTransferCrossChainAsset:
		for k, v := range payloadObj.AddressesMap {
			info := &DepositInfo{
				MainChainProgramHash: txn.Outputs[v].ProgramHash,
				TargetAddress:        k,
				Amount:               txn.Outputs[v].Value,
			}
			result = append(result, info)
		}
	default:
		return nil, errors.New("Invalid payload")
	}

	return result, nil
}

func (mc *MainChainImpl) syncChainData() {
	var chainHeight uint32
	var currentHeight uint32
	var needSync bool

	for {
		chainHeight, currentHeight, needSync = mc.needSyncBlocks()
		if !needSync {
			break
		}

		for currentHeight < chainHeight {
			block, err := rpc.GetBlockByHeight(currentHeight, config.Parameters.MainNode.Rpc)
			if err != nil {
				break
			}
			mc.processBlock(block)

			// Update wallet height
			currentHeight = DbCache.CurrentHeight(block.Height + 1)

			fmt.Print(">")
		}
	}

	fmt.Print("\n")
}

func (mc *MainChainImpl) needSyncBlocks() (uint32, uint32, bool) {

	chainHeight, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return 0, 0, false
	}

	currentHeight := DbCache.CurrentHeight(QueryHeightCode)

	if currentHeight >= chainHeight {
		return chainHeight, currentHeight, false
	}

	return chainHeight, currentHeight, true
}

func (mc *MainChainImpl) getAvailableUTXOs(utxos []*AddressUTXO) []*AddressUTXO {
	var availableUTXOs []*AddressUTXO
	var currentHeight = DbCache.CurrentHeight(QueryHeightCode)
	for _, utxo := range utxos {
		if utxo.Input.Sequence > 0 {
			if utxo.Input.Sequence >= currentHeight {
				continue
			}
			utxo.Input.Sequence = math.MaxUint32 - 1
		}
		availableUTXOs = append(availableUTXOs, utxo)
	}
	return availableUTXOs
}

func (mc *MainChainImpl) containGenesisBlockAddress(address string) (string, bool) {
	for _, node := range config.Parameters.SideNodeList {
		if node.GenesisBlockAddress == address {
			return node.DestroyAddress, true
		}
	}
	return "", false
}

func (mc *MainChainImpl) processBlock(block *BlockInfo) {
	// Add UTXO to wallet address from transaction outputs
	for _, txnInfo := range block.Tx {
		var txn TransactionInfo
		rpc.Unmarshal(&txnInfo, &txn)

		// Add UTXOs to wallet address from transaction outputs
		for index, output := range txn.Outputs {
			if destroyAddress, ok := mc.containGenesisBlockAddress(output.Address); ok {
				// Create UTXO input from output
				txHashBytes, _ := HexStringToBytes(txn.Hash)
				//txHashBytes = BytesReverse(txHashBytes)
				referTxHash, _ := Uint256FromBytes(txHashBytes)
				sequence := output.OutputLock
				if txn.TxType == CoinBase {
					sequence = block.Height + 100
				}
				input := &Input{
					Previous: OutPoint{
						TxID:  *referTxHash,
						Index: uint16(index),
					},
					Sequence: sequence,
				}
				amount, _ := StringToFixed64(output.Value)
				// Save UTXO input to data store
				addressUTXO := &AddressUTXO{
					Input:               input,
					Amount:              amount,
					GenesisBlockAddress: output.Address,
					DestroyAddress:      destroyAddress,
				}
				DbCache.AddAddressUTXO(addressUTXO)
			}
		}

		// Delete UTXOs from wallet by transaction inputs
		for _, input := range txn.Inputs {
			txHashBytes, _ := HexStringToBytes(input.TxID)
			txHashBytes = BytesReverse(txHashBytes)
			referTxID, _ := Uint256FromBytes(txHashBytes)
			txInput := &Input{
				Previous: OutPoint{
					TxID:  *referTxID,
					Index: input.VOut,
				},
				Sequence: input.Sequence,
			}
			DbCache.DeleteUTXO(txInput)
		}
	}
}

func InitMainChain(arbitrator Arbitrator) error {
	currentArbitrator, ok := arbitrator.(*ArbitratorImpl)
	if !ok {
		return errors.New("Unknown arbitrator type.")
	}

	mainChainServer := &MainChainImpl{&DistributedNodeServer{P2pCommand: WithdrawCommand}}
	P2PClientSingleton.AddListener(mainChainServer)
	currentArbitrator.SetMainChain(mainChainServer)

	mainChainClient := &MainChainClientImpl{&DistributedNodeClient{P2pCommand: WithdrawCommand}}
	P2PClientSingleton.AddListener(mainChainClient)
	currentArbitrator.SetMainChainClient(mainChainClient)

	return nil
}
