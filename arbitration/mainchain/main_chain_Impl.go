package mainchain

import (
	"errors"
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

const WithdrawFromSideChainLockTime uint32 = 6

type MainChainImpl struct {
	*DistributedNodeServer
}

func (mc *MainChainImpl) SyncMainChainCachedTxs() (map[SideChain][]string, error) {
	txHases, genesisAddresses, transactions, _, err := DbCache.GetAllMainChainTxs()
	if err != nil {
		return nil, err
	}

	if len(txHases) != len(transactions) {
		return nil, errors.New("Invalid transactios in main chain txs db")
	}

	allSideChainTxHashes := make(map[SideChain][]string, 0)
	for i := 0; i < len(transactions); i++ {
		sc, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(genesisAddresses[i])
		if !ok {
			log.Warn("Invalid deposit address.")
			continue
		}

		hasSideChainInMap := false
		for k, _ := range allSideChainTxHashes {
			if k == sc {
				hasSideChainInMap = true
				break
			}
		}
		if hasSideChainInMap {
			allSideChainTxHashes[sc] = append(allSideChainTxHashes[sc], txHases[i])
		} else {
			allSideChainTxHashes[sc] = []string{txHases[i]}
		}
	}

	result := make(map[SideChain][]string, 0)
	for k, v := range allSideChainTxHashes {
		receivedTxs, err := k.GetExistDepositTransactions(v)
		if err != nil {
			log.Warn("Invalid deposit address.")
			continue
		}
		unsolvedTxs := SubstractTransactionHashes(v, receivedTxs)
		result[k] = unsolvedTxs

		for _, recTx := range receivedTxs {
			err = DbCache.RemoveMainChainTx(recTx, k.GetKey())
			if err != nil {
				return nil, err
			}
			err = FinishedTxsDbCache.AddSucceedDepositTx(recTx, k.GetKey())
			if err != nil {
				log.Error("Add succeed deposit transactions into finished db failed")
			}
		}
	}

	return result, err
}

func (mc *MainChainImpl) OnP2PReceived(peer *net.Peer, msg p2p.Message) error {
	if msg.CMD() != mc.P2pCommand {
		return nil
	}

	switch m := msg.(type) {
	case *SignMessage:
		return mc.ReceiveProposalFeedback(m.Content)
	}
	return nil
}

func (mc *MainChainImpl) CreateWithdrawTransaction(sideChain SideChain, withdrawInfo *WithdrawInfo,
	sideChainTransactionHashes []string, mcFunc MainChainFunc) (*Transaction, error) {

	mc.SyncChainData()

	withdrawBank := sideChain.GetKey()
	rate := sideChain.GetExchangeRate()

	var totalOutputAmount Fixed64
	// Create transaction outputs
	var txOutputs []*Output
	// Check if from address is valid
	assetID := spvWallet.SystemAssetId
	for i := 0; i < len(withdrawInfo.TargetAddress); i++ {
		amount := withdrawInfo.Amount[i] / Fixed64(rate)
		crossChainAmount := withdrawInfo.CrossChainAmount[i] / Fixed64(rate)
		programhash, err := Uint168FromAddress(withdrawInfo.TargetAddress[i])
		if err != nil {
			return nil, err
		}

		txOutput := &Output{
			AssetID:     Uint256(assetID),
			ProgramHash: *programhash,
			Value:       crossChainAmount,
			OutputLock:  uint32(WithdrawFromSideChainLockTime),
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
	sideChain.AddLastUsedOutPoints(newUsedUtxos)

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

	txPayload := &PayloadWithdrawFromSideChain{
		BlockHeight:                chainHeight,
		GenesisBlockAddress:        withdrawBank,
		SideChainTransactionHashes: sideChainTransactionHashes}
	program := &Program{redeemScript, nil}

	// Create attributes
	txAttr := NewAttribute(Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	attributes := make([]*Attribute, 0)
	attributes = append(attributes, &txAttr)

	return &Transaction{
		TxType:     WithdrawFromSideChain,
		Payload:    txPayload,
		Attributes: attributes,
		Inputs:     txInputs,
		Outputs:    txOutputs,
		Programs:   []*Program{program},
		LockTime:   uint32(0),
	}, nil
}

func (mc *MainChainImpl) ParseUserDepositTransactionInfo(txn *Transaction, genesisAddress string) (*DepositInfo, error) {
	result := new(DepositInfo)
	payloadObj, ok := txn.Payload.(*PayloadTransferCrossChainAsset)
	if !ok {
		return nil, errors.New("Invalid payload")
	}
	if len(txn.Outputs) == 0 {
		return nil, errors.New("Invalid TransferCrossChainAsset payload, outputs is null")
	}
	programHash, err := Uint168FromAddress(genesisAddress)
	if err != nil {
		return nil, errors.New("Invalid genesis address")
	}
	result.MainChainProgramHash = *programHash
	for i := 0; i < len(payloadObj.CrossChainAddress); i++ {
		if txn.Outputs[payloadObj.OutputIndex[i]].ProgramHash == result.MainChainProgramHash {
			result.TargetAddress = append(result.TargetAddress, payloadObj.CrossChainAddress[i])
			result.Amount = append(result.Amount, txn.Outputs[payloadObj.OutputIndex[i]].Value)
			result.CrossChainAmount = append(result.CrossChainAmount, payloadObj.CrossChainAmount[i])
		}
	}

	return result, nil
}

func (mc *MainChainImpl) SyncChainData() {
	var chainHeight uint32
	var currentHeight uint32
	var needSync bool

	for {
		chainHeight, currentHeight, needSync = mc.needSyncBlocks()
		if !needSync {
			break
		}

		for currentHeight < chainHeight {
			block, err := rpc.GetBlockByHeight(currentHeight+1, config.Parameters.MainNode.Rpc)
			if err != nil {
				break
			}
			mc.processBlock(block, currentHeight+1)

			// Update wallet height
			currentHeight = DbCache.CurrentHeight(block.Height)
			log.Info("[arbitrator] Main chain height: ", block.Height)
		}
	}
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

func (mc *MainChainImpl) processBlock(block *BlockInfo, height uint32) {
	sideChains := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetAllChains()
	// Add UTXO to wallet address from transaction outputs
	for _, txnInfo := range block.Tx {
		var txn TransactionInfo
		rpc.Unmarshal(&txnInfo, &txn)

		// Add UTXOs to wallet address from transaction outputs
		for index, output := range txn.Outputs {
			if ok := mc.containGenesisBlockAddress(output.Address); ok {
				// Create UTXO input from output
				txHashBytes, _ := HexStringToBytes(txn.Hash)
				referTxHash, _ := Uint256FromBytes(BytesReverse(txHashBytes))
				sequence := output.OutputLock
				input := &Input{
					Previous: OutPoint{
						TxID:  *referTxHash,
						Index: uint16(index),
					},
					Sequence: sequence,
				}
				if txn.TxType == CoinBase {
					sequence = block.Height + 100
				}

				amount, _ := StringToFixed64(output.Value)
				// Save UTXO input to data store
				addressUTXO := &AddressUTXO{
					Input:               input,
					Amount:              amount,
					GenesisBlockAddress: output.Address,
				}
				if *amount > Fixed64(0) {
					DbCache.AddAddressUTXO(addressUTXO)
				}
			}
		}

		// Delete UTXOs from wallet by transaction inputs
		for _, input := range txn.Inputs {
			txHashBytes, _ := HexStringToBytes(input.TxID)
			referTxID, _ := Uint256FromBytes(BytesReverse(txHashBytes))
			outPoint := OutPoint{
				TxID:  *referTxID,
				Index: input.VOut,
			}
			txInput := &Input{
				Previous: outPoint,
				Sequence: input.Sequence,
			}
			DbCache.DeleteUTXO(txInput)

			for _, sc := range sideChains {
				var containedOps []OutPoint
				for _, op := range sc.GetLastUsedOutPoints() {
					if op.IsEqual(outPoint) {
						containedOps = append(containedOps, op)
					}
				}
				if len(containedOps) != 0 {
					sc.RemoveLastUsedOutPoints(containedOps)
				}
			}
		}
	}

	for _, sc := range sideChains {
		sc.SetLastUsedUtxoHeight(height)
		log.Info("Side chain [", sc.GetKey(), "] SetLastUsedUtxoHeight ", height)
	}
}

func (mc *MainChainImpl) CheckAndRemoveDepositTransactionsFromDB() error {
	//remove deposit transactions if exist on side chain
	txHases, genesisAddresses, transactions, _, err := DbCache.GetAllMainChainTxs()
	if err != nil {
		return err
	}

	if len(txHases) != len(transactions) {
		return errors.New("Invalid transactios in main chain txs db")
	}

	allSideChainTxHashes := make(map[SideChain][]string, 0)
	for i := 0; i < len(transactions); i++ {
		sc, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(genesisAddresses[i])
		if !ok {
			log.Warn("Invalid deposit address.")
			continue
		}

		hasSideChainInMap := false
		for k, _ := range allSideChainTxHashes {
			if k == sc {
				hasSideChainInMap = true
				break
			}
		}
		if hasSideChainInMap {
			allSideChainTxHashes[sc] = append(allSideChainTxHashes[sc], txHases[i])
		} else {
			allSideChainTxHashes[sc] = []string{txHases[i]}
		}
	}

	for k, v := range allSideChainTxHashes {
		receivedTxs, err := k.GetExistDepositTransactions(v)
		if err != nil {
			log.Warn("Invalid deposit address.")
			continue
		}
		for _, recTx := range receivedTxs {
			err = DbCache.RemoveMainChainTx(recTx, k.GetKey())
			if err != nil {
				return err
			}
			err = FinishedTxsDbCache.AddSucceedDepositTx(recTx, k.GetKey())
			if err != nil {
				log.Error("Add succeed deposit transactions into finished db failed")
			}
		}
	}

	return nil
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
