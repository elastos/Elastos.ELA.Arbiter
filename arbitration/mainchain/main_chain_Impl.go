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

	. "github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	. "github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/p2p"
	"github.com/elastos/Elastos.ELA/p2p/peer"
)

type MainChainImpl struct {
	*DistributedNodeServer
}

func (mc *MainChainImpl) SyncMainChainCachedTxs() error {
	log.Info("[SyncMainChainCachedTxs] start")
	defer log.Info("[SyncMainChainCachedTxs] end")

	txs, err := DbCache.MainChainStore.GetAllMainChainTxs()
	if err != nil {
		return errors.New("[SyncMainChainCachedTxs]" + err.Error())
	}

	if len(txs) == 0 {
		return errors.New("[SyncMainChainCachedTxs] No main chain tx in dbcache")
	}

	allSideChainTxHashes := make(map[SideChain][]string, 0)
	for _, tx := range txs {
		sc, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(tx.GenesisBlockAddress)
		if !ok {
			log.Warn("[SyncMainChainCachedTxs] Get side chain from genesis address failed")
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
			allSideChainTxHashes[sc] = append(allSideChainTxHashes[sc], tx.TransactionHash)
		} else {
			allSideChainTxHashes[sc] = []string{tx.TransactionHash}
		}
	}

	for k, v := range allSideChainTxHashes {
		go mc.createAndSendDepositTransactionsInDB(k, v)
	}

	return nil
}

func (mc *MainChainImpl) createAndSendDepositTransactionsInDB(sideChain SideChain, txHashes []string) {
	receivedTxs, err := sideChain.GetExistDepositTransactions(txHashes)
	if err != nil {
		log.Warn("[SyncMainChainCachedTxs] Get exist deposit transactions failed, err:", err.Error())
		return
	}
	unsolvedTxs := SubstractTransactionHashes(txHashes, receivedTxs)
	var addresses []string
	for i := 0; i < len(receivedTxs); i++ {
		addresses = append(addresses, sideChain.GetKey())
	}
	err = DbCache.MainChainStore.RemoveMainChainTxs(receivedTxs, addresses)
	if err != nil {
		log.Warn("[SyncMainChainCachedTxs] Remove main chain txs failed, err:", err.Error())
	}
	err = FinishedTxsDbCache.AddSucceedDepositTxs(receivedTxs, addresses)
	if err != nil {
		log.Error("[SyncMainChainCachedTxs] Add succeed deposit transactions into finished db failed, err:", err.Error())
	}

	spvTxs, err := DbCache.MainChainStore.GetMainChainTxsFromHashes(unsolvedTxs, sideChain.GetKey())
	if err != nil {
		log.Error("[SyncMainChainCachedTxs] Get main chain txs from hashes failed, err:", err.Error())
		return
	}

	ArbitratorGroupSingleton.GetCurrentArbitrator().SendDepositTransactions(spvTxs, sideChain.GetKey())
}

func (mc *MainChainImpl) OnP2PReceived(peer *peer.Peer, msg p2p.Message) error {
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

	withdrawBank := sideChain.GetKey()
	exchangeRate, err := sideChain.GetExchangeRate()
	if err != nil {
		return nil, err
	}

	var totalOutputAmount Fixed64
	// Create transaction outputs
	var txOutputs []*Output
	// Check if from address is valid
	assetID := SystemAssetId
	for _, withdraw := range withdrawInfo.WithdrawAssets {
		programhash, err := Uint168FromAddress(withdraw.TargetAddress)
		if err != nil {
			return nil, err
		}
		txOutput := &Output{
			AssetID:     Uint256(assetID),
			ProgramHash: *programhash,
			Value:       Fixed64(float64(*withdraw.CrossChainAmount) / exchangeRate),
			OutputLock:  0,
		}
		txOutputs = append(txOutputs, txOutput)
		totalOutputAmount += Fixed64(float64(*withdraw.Amount) / exchangeRate)
	}

	availableUTXOs, err := mcFunc.GetAvailableUtxos(withdrawBank)
	if err != nil {
		return nil, err
	}

	//get real available utxos
	ops := sideChain.GetLastUsedOutPoints()

	var realAvailableUtxos []*AddressUTXO
	var unavailableUtxos []*AddressUTXO
	for _, utxo := range availableUTXOs {
		isUsed := false
		for _, ops := range ops {
			if ops.IsEqual(utxo.Input.Previous) {
				isUsed = true
			}
		}
		if !isUsed {
			realAvailableUtxos = append(realAvailableUtxos, utxo)
		} else {
			unavailableUtxos = append(unavailableUtxos, utxo)
		}
	}

	// Create transaction inputs
	var txInputs []*Input
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
				AssetID:     Uint256(SystemAssetId),
				Value:       Fixed64(*utxo.Amount - totalOutputAmount),
				OutputLock:  uint32(0),
				ProgramHash: *programHash,
			}
			txOutputs = append(txOutputs, change)
			totalOutputAmount = 0
			break
		}
	}

	//if available utxo is not enough, try to use unavailable utxos from biggest one
	if totalOutputAmount > 0 && len(unavailableUtxos) != 0 {
		for i := len(unavailableUtxos) - 1; i >= 0; i++ {
			utxo := unavailableUtxos[i]
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
					AssetID:     Uint256(SystemAssetId),
					Value:       Fixed64(*utxo.Amount - totalOutputAmount),
					OutputLock:  uint32(0),
					ProgramHash: *programHash,
				}
				txOutputs = append(txOutputs, change)
				totalOutputAmount = 0
				break
			}
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

	var txHashes []Uint256
	for _, hash := range sideChainTransactionHashes {
		hashBytes, err := HexStringToBytes(hash)
		if err != nil {
			return nil, err
		}
		txHash, err := Uint256FromBytes(hashBytes)
		if err != nil {
			return nil, err
		}
		txHashes = append(txHashes, *txHash)
	}

	txPayload := &payload.PayloadWithdrawFromSideChain{
		BlockHeight:                chainHeight,
		GenesisBlockAddress:        withdrawBank,
		SideChainTransactionHashes: txHashes}
	p := &program.Program{redeemScript, nil}

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
		Programs:   []*program.Program{p},
		LockTime:   uint32(0),
	}, nil
}

func (mc *MainChainImpl) SyncChainData() {
	var chainHeight uint32
	var currentHeight uint32
	var needSync bool

	for {
		chainHeight, currentHeight, needSync = mc.needSyncBlocks()
		if !needSync {
			log.Debug("No need sync, chain height:", chainHeight, "current height:", currentHeight)
			break
		}
		log.Info("[arbitrator] Main chain height: ", chainHeight)

		//sync genesis block
		if currentHeight == 0 {
			err := mc.syncAndProcessBlock(currentHeight)
			if err != nil {
				log.Error("get genesis block failed, chainHeight:", chainHeight)
				break
			}
		}

		for currentHeight < chainHeight {
			err := mc.syncAndProcessBlock(currentHeight + 1)
			if err != nil {
				log.Error("get block by height failed, chain height:", chainHeight,
					"current height:", currentHeight+1, "err:", err.Error())
				break
			}
			currentHeight += 1
		}
		// Update wallet height
		currentHeight = DbCache.UTXOStore.CurrentHeight(currentHeight)
	}
}

func (mc *MainChainImpl) syncAndProcessBlock(currentHeight uint32) error {
	block, err := rpc.GetBlockByHeight(currentHeight, config.Parameters.MainNode.Rpc)
	if err != nil {
		return err
	}

	mc.processBlock(block, currentHeight)
	return nil
}

func (mc *MainChainImpl) needSyncBlocks() (uint32, uint32, bool) {

	chainHeight, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return 0, 0, false
	}

	currentHeight := DbCache.UTXOStore.CurrentHeight(QueryHeightCode)

	if currentHeight >= chainHeight {
		return chainHeight, currentHeight, false
	}

	return chainHeight, currentHeight, true
}

func (mc *MainChainImpl) getAvailableUTXOs(utxos []*AddressUTXO) []*AddressUTXO {
	var availableUTXOs []*AddressUTXO
	var currentHeight = DbCache.UTXOStore.CurrentHeight(QueryHeightCode)
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
	log.Info("[processBlock] block height:", block.Height, "current height:", height)
	sideChains := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetAllChains()
	// Add UTXO to wallet address from transaction outputs
	utxos := make([]*AddressUTXO, 0)
	inputs := make([]*Input, 0)
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
					utxos = append(utxos, addressUTXO)
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
			inputs = append(inputs, txInput)
		}
	}
	DbCache.UTXOStore.AddAddressUTXOs(utxos)
	DbCache.UTXOStore.DeleteUTXOs(inputs)

	for _, sc := range sideChains {
		sc.ClearLastUsedOutPoints()
		sc.SetLastUsedUtxoHeight(height)
		log.Info("Side chain [", sc.GetKey(), "] SetLastUsedUtxoHeight ", height)
	}
}

func (mc *MainChainImpl) CheckAndRemoveDepositTransactionsFromDB() error {
	//remove deposit transactions if exist on side chain
	txs, err := DbCache.MainChainStore.GetAllMainChainTxs()
	if err != nil {
		return err
	}

	if len(txs) == 0 {
		return nil
	}

	allSideChainTxHashes := make(map[SideChain][]string, 0)
	for _, tx := range txs {
		sc, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(tx.GenesisBlockAddress)
		if !ok {
			log.Warn("[CheckAndRemoveDepositTransactionsFromDB] Get chain from genesis addres failed.")
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
			allSideChainTxHashes[sc] = append(allSideChainTxHashes[sc], tx.TransactionHash)
		} else {
			allSideChainTxHashes[sc] = []string{tx.TransactionHash}
		}
	}

	for k, v := range allSideChainTxHashes {
		receivedTxs, err := k.GetExistDepositTransactions(v)
		if err != nil {
			log.Warn("[CheckAndRemoveDepositTransactionsFromDB] Get exist deposit transactions failed:", err.Error())
			continue
		}
		finalGenesisAddresses := make([]string, 0)
		for i := 0; i < len(receivedTxs); i++ {
			finalGenesisAddresses = append(finalGenesisAddresses, k.GetKey())
		}
		err = DbCache.MainChainStore.RemoveMainChainTxs(receivedTxs, finalGenesisAddresses)
		if err != nil {
			return err
		}
		err = FinishedTxsDbCache.AddSucceedDepositTxs(receivedTxs, finalGenesisAddresses)
		if err != nil {
			log.Error("[CheckAndRemoveDepositTransactionsFromDB] Add succeed deposit transactions into finished db failed")
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
