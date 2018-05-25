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
	"github.com/elastos/Elastos.ELA/core"
	. "github.com/elastos/Elastos.ELA/core"
)

const WithdrawAssetLockTime uint32 = 6

type MainChainImpl struct {
	*DistributedNodeServer
}

func (mc *MainChainImpl) SyncMainChainCachedTxs() (map[SideChain][]string, error) {
	txHases, err := DbCache.GetAllMainChainTxHashes()
	if err != nil {
		return nil, err
	}

	transactions, _, err := DbCache.GetMainChainTxsFromHashes(txHases)
	if err != nil {
		return nil, err
	}

	if len(txHases) != len(transactions) {
		return nil, errors.New("Invalid transactios in main chain txs db")
	}

	allSideChainTxHashes := make(map[SideChain][]string, 0)
	for i := 0; i < len(transactions); i++ {
		depositInfo, err := mc.ParseUserDepositTransactionInfo(transactions[i])
		if err != nil || len(depositInfo) == 0 {
			log.Warn("Invalid deposit address.")
			continue
		}

		addr, err := depositInfo[0].MainChainProgramHash.ToAddress()
		if err != nil {
			log.Warn("Invalid deposit address.")
			continue
		}
		sc, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(addr)
		if !ok {
			log.Warn("Invalid deposit address.")
			continue
		}

		hasSideChainInMap := false
		for k, _ := range allSideChainTxHashes {
			if k == sc {
				hasSideChainInMap = true
			}
		}
		if hasSideChainInMap {
			allSideChainTxHashes[sc] = append(allSideChainTxHashes[sc], txHases[i])
		} else {
			allSideChainTxHashes[sc] = []string{txHases[i]}
		}
	}

	result := make(map[SideChain][]string, 0)
	var receivedTxs []string
	for k, v := range allSideChainTxHashes {
		recTxs, err := k.GetExistDepositTransactions(v)
		if err != nil {
			log.Warn("Invalid deposit address.")
			continue
		}
		unsolvedTxs := SubstractTransactionHashes(v, recTxs)
		result[k] = unsolvedTxs

		for _, recTx := range recTxs {
			receivedTxs = append(receivedTxs, recTx)
		}
	}
	err = DbCache.RemoveMainChainTxs(receivedTxs)
	if err != nil {
		return nil, err
	}

	if len(receivedTxs) != 0 {
		mc.SendTxCacheClearMessage(receivedTxs)
	}

	return result, err
}

func (mc *MainChainImpl) SendTxCacheClearMessage(receivedTxs []string) {
	msg := &TxCacheClearMessage{Command: DepositTxCacheClearCommand, RemovedTxs: receivedTxs}
	P2PClientSingleton.AddMessageHash(P2PClientSingleton.GetMessageHash(msg))
	P2PClientSingleton.Broadcast(msg)
}

func (mc *MainChainImpl) OnP2PReceived(peer *net.Peer, msg p2p.Message) error {
	if msg.CMD() != mc.P2pCommand && msg.CMD() != WithdrawTxCacheClearCommand {
		return nil
	}

	switch m := msg.(type) {
	case *SignMessage:
		return mc.ReceiveProposalFeedback(m.Content)
	case *TxCacheClearMessage:
		return DbCache.RemoveSideChainTxs(m.RemovedTxs)
	}
	return nil
}

func (mc *MainChainImpl) CreateWithdrawTransaction(sideChain SideChain, infoArray []*WithdrawInfo,
	sideChainTransactionHash []string, mcFunc MainChainFunc) (*Transaction, error) {

	mc.syncChainData()

	withdrawBank := sideChain.GetKey()
	rate := sideChain.GetRage()

	var totalOutputAmount Fixed64
	// Create transaction outputs
	var txOutputs []*Output
	// Check if from address is valid
	assetID := spvWallet.SystemAssetId
	for _, info := range infoArray {
		amount := info.Amount / Fixed64(rate)
		crossChainAmount := info.CrossChainAmount / Fixed64(rate)
		programhash, err := Uint168FromAddress(info.TargetAddress)
		if err != nil {
			return nil, err
		}

		txOutput := &Output{
			AssetID:     Uint256(assetID),
			ProgramHash: *programhash,
			Value:       crossChainAmount,
			OutputLock:  uint32(WithdrawAssetLockTime),
		}

		txOutputs = append(txOutputs, txOutput)
		totalOutputAmount += amount
	}

	var txInputs []*Input
	availableUTXOs, err := mcFunc.GetAvailableUtxos(withdrawBank)
	if err != nil {
		return nil, err
	}

	//get real available utxos
	ops := sideChain.GetLastUsedOutPoints()

	var realAvailableUtxos []*AddressUTXO
	for _, utxo := range availableUTXOs {
		isUsed := false
		for _, ops := range ops {
			if ops.IsEqual(utxo.Input.Previous) {
				isUsed = true
			}
		}
		if !isUsed {
			realAvailableUtxos = append(realAvailableUtxos, utxo)
		}
	}

	// Create transaction inputs
	for _, utxo := range realAvailableUtxos {
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

	var newUsedUtxos []core.OutPoint
	for _, input := range txInputs {
		newUsedUtxos = append(newUsedUtxos, input.Previous)
	}
	sideChain.SetLastUsedOutPoints(newUsedUtxos)

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
		for i := 0; i < len(payloadObj.CrossChainAddress); i++ {
			info := &DepositInfo{
				MainChainProgramHash: txn.Outputs[payloadObj.OutputIndex[i]].ProgramHash,
				TargetAddress:        payloadObj.CrossChainAddress[i],
				Amount:               txn.Outputs[payloadObj.OutputIndex[i]].Value,
				CrossChainAmount:     payloadObj.CrossChainAmount[i],
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

func (mc *MainChainImpl) containGenesisBlockAddress(address string) bool {
	for _, node := range config.Parameters.SideNodeList {
		if node.GenesisBlockAddress == address {
			return true
		}
	}
	return false
}

func (mc *MainChainImpl) processBlock(block *BlockInfo) {
	// Add UTXO to wallet address from transaction outputs
	for _, txnInfo := range block.Tx {
		var txn TransactionInfo
		rpc.Unmarshal(&txnInfo, &txn)

		// Add UTXOs to wallet address from transaction outputs
		for index, output := range txn.Outputs {
			if ok := mc.containGenesisBlockAddress(output.Address); ok {
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
					DestroyAddress:      DESTROY_ADDRESS,
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
