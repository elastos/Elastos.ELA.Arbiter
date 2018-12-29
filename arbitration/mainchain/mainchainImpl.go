package mainchain

import (
	"errors"
	"math"
	"math/rand"
	"strconv"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	peer2 "github.com/elastos/Elastos.ELA/dpos/p2p/peer"
)

type MainChainImpl struct {
	*cs.DistributedNodeServer
}

func (mc *MainChainImpl) SyncMainChainCachedTxs() error {
	log.Info("[SyncMainChainCachedTxs] start")
	defer log.Info("[SyncMainChainCachedTxs] end")

	txs, err := store.DbCache.MainChainStore.GetAllMainChainTxs()
	if err != nil {
		return errors.New("[SyncMainChainCachedTxs]" + err.Error())
	}

	if len(txs) == 0 {
		return errors.New("[SyncMainChainCachedTxs] No main chain tx in dbcache")
	}

	allSideChainTxHashes := make(map[arbitrator.SideChain][]string, 0)
	for _, tx := range txs {
		sc, ok := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(tx.GenesisBlockAddress)
		if !ok {
			log.Warn("[SyncMainChainCachedTxs] Get side chain from genesis address failed")
			continue
		}

		hasSideChainInMap := false
		for k := range allSideChainTxHashes {
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

func (mc *MainChainImpl) createAndSendDepositTransactionsInDB(sideChain arbitrator.SideChain, txHashes []string) {
	receivedTxs, err := sideChain.GetExistDepositTransactions(txHashes)
	if err != nil {
		log.Warn("[SyncMainChainCachedTxs] Get exist deposit transactions failed, err:", err.Error())
		return
	}
	unsolvedTxs := base.SubstractTransactionHashes(txHashes, receivedTxs)
	var addresses []string
	for i := 0; i < len(receivedTxs); i++ {
		addresses = append(addresses, sideChain.GetKey())
	}
	err = store.DbCache.MainChainStore.RemoveMainChainTxs(receivedTxs, addresses)
	if err != nil {
		log.Warn("[SyncMainChainCachedTxs] Remove main chain txs failed, err:", err.Error())
	}
	err = store.FinishedTxsDbCache.AddSucceedDepositTxs(receivedTxs, addresses)
	if err != nil {
		log.Error("[SyncMainChainCachedTxs] Add succeed deposit transactions into finished db failed, err:", err.Error())
	}

	spvTxs, err := store.DbCache.MainChainStore.GetMainChainTxsFromHashes(unsolvedTxs, sideChain.GetKey())
	if err != nil {
		log.Error("[SyncMainChainCachedTxs] Get main chain txs from hashes failed, err:", err.Error())
		return
	}

	arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().SendDepositTransactions(spvTxs, sideChain.GetKey())
}

func (mc *MainChainImpl) OnReceivedSignMsg(id peer2.PID, content []byte) {
	if err := mc.ReceiveProposalFeedback(content); err != nil {
		log.Error("[OnReceivedSignMsg] mainchain received distributed item message error: ", err)
	}
}

func (mc *MainChainImpl) CreateWithdrawTransaction(sideChain arbitrator.SideChain, withdrawInfo *base.WithdrawInfo,
	sideChainTransactionHashes []string, mcFunc arbitrator.MainChainFunc) (*types.Transaction, error) {

	withdrawBank := sideChain.GetKey()
	exchangeRate, err := sideChain.GetExchangeRate()
	if err != nil {
		return nil, err
	}

	var totalOutputAmount common.Fixed64
	// Create transaction outputs
	var txOutputs []*types.Output
	// Check if from address is valid
	assetID := base.SystemAssetId
	for _, withdraw := range withdrawInfo.WithdrawAssets {
		programhash, err := common.Uint168FromAddress(withdraw.TargetAddress)
		if err != nil {
			return nil, err
		}
		txOutput := &types.Output{
			AssetID:     common.Uint256(assetID),
			ProgramHash: *programhash,
			Value:       common.Fixed64(float64(*withdraw.CrossChainAmount) / exchangeRate),
			OutputLock:  0,
		}
		txOutputs = append(txOutputs, txOutput)
		totalOutputAmount += common.Fixed64(float64(*withdraw.Amount) / exchangeRate)
	}

	availableUTXOs, err := mcFunc.GetAvailableUtxos(withdrawBank)
	if err != nil {
		return nil, err
	}

	//get real available utxos
	ops := sideChain.GetLastUsedOutPoints()

	var realAvailableUtxos []*store.AddressUTXO
	var unavailableUtxos []*store.AddressUTXO
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
	var txInputs []*types.Input
	for _, utxo := range realAvailableUtxos {
		txInputs = append(txInputs, utxo.Input)
		if *utxo.Amount < totalOutputAmount {
			totalOutputAmount -= *utxo.Amount
		} else if *utxo.Amount == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if *utxo.Amount > totalOutputAmount {
			programHash, err := common.Uint168FromAddress(withdrawBank)
			if err != nil {
				return nil, err
			}
			change := &types.Output{
				AssetID:     common.Uint256(base.SystemAssetId),
				Value:       common.Fixed64(*utxo.Amount - totalOutputAmount),
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
				programHash, err := common.Uint168FromAddress(withdrawBank)
				if err != nil {
					return nil, err
				}
				change := &types.Output{
					AssetID:     common.Uint256(base.SystemAssetId),
					Value:       common.Fixed64(*utxo.Amount - totalOutputAmount),
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

	redeemScript, err := cs.CreateRedeemScript()
	if err != nil {
		return nil, err
	}

	// Create payload
	chainHeight, err := mcFunc.GetMainNodeCurrentHeight()
	if err != nil {
		return nil, err
	}

	var txHashes []common.Uint256
	for _, hash := range sideChainTransactionHashes {
		hashBytes, err := common.HexStringToBytes(hash)
		if err != nil {
			return nil, err
		}
		txHash, err := common.Uint256FromBytes(hashBytes)
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

	// Create attribute
	txAttr := types.NewAttribute(types.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	attributes := make([]*types.Attribute, 0)
	attributes = append(attributes, &txAttr)

	return &types.Transaction{
		TxType:     types.WithdrawFromSideChain,
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
	}
}

func (mc *MainChainImpl) syncAndProcessBlock(currentHeight uint32) error {
	block, err := rpc.GetBlockByHeight(currentHeight, config.Parameters.MainNode.Rpc)
	if err != nil {
		return err
	}
	mc.processBlock(block, currentHeight)

	// Update wallet height
	currentHeight = store.DbCache.UTXOStore.CurrentHeight(block.Height)

	cs.P2PClientSingleton.ChangeHeight(currentHeight)

	return nil
}

func (mc *MainChainImpl) needSyncBlocks() (uint32, uint32, bool) {

	chainHeight, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return 0, 0, false
	}

	currentHeight := store.DbCache.UTXOStore.CurrentHeight(store.QueryHeightCode)

	if currentHeight >= chainHeight {
		return chainHeight, currentHeight, false
	}

	return chainHeight, currentHeight, true
}

func (mc *MainChainImpl) getAvailableUTXOs(utxos []*store.AddressUTXO) []*store.AddressUTXO {
	var availableUTXOs []*store.AddressUTXO
	var currentHeight = store.DbCache.UTXOStore.CurrentHeight(store.QueryHeightCode)
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

func (mc *MainChainImpl) processBlock(block *base.BlockInfo, height uint32) {
	log.Info("[processBlock] block height:", block.Height, "current height:", height)
	sideChains := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetAllChains()
	// Add UTXO to wallet address from transaction outputs
	for _, txnInfo := range block.Tx {
		var txn base.TransactionInfo
		rpc.Unmarshal(&txnInfo, &txn)

		// Add UTXOs to wallet address from transaction outputs
		for index, output := range txn.Outputs {
			if ok := mc.containGenesisBlockAddress(output.Address); ok {
				// Create UTXO input from output
				txHashBytes, _ := common.HexStringToBytes(txn.Hash)
				referTxHash, _ := common.Uint256FromBytes(common.BytesReverse(txHashBytes))
				sequence := output.OutputLock
				input := &types.Input{
					Previous: types.OutPoint{
						TxID:  *referTxHash,
						Index: uint16(index),
					},
					Sequence: sequence,
				}
				if txn.TxType == types.CoinBase {
					sequence = block.Height + 100
				}

				amount, _ := common.StringToFixed64(output.Value)
				// Save UTXO input to data store
				addressUTXO := &store.AddressUTXO{
					Input:               input,
					Amount:              amount,
					GenesisBlockAddress: output.Address,
				}
				if *amount > common.Fixed64(0) {
					store.DbCache.UTXOStore.AddAddressUTXO(addressUTXO)
				}
			}
		}

		// Delete UTXOs from wallet by transaction inputs
		for _, input := range txn.Inputs {
			txHashBytes, _ := common.HexStringToBytes(input.TxID)
			referTxID, _ := common.Uint256FromBytes(common.BytesReverse(txHashBytes))
			outPoint := types.OutPoint{
				TxID:  *referTxID,
				Index: input.VOut,
			}
			txInput := &types.Input{
				Previous: outPoint,
				Sequence: input.Sequence,
			}
			store.DbCache.UTXOStore.DeleteUTXO(txInput)
		}
	}

	for _, sc := range sideChains {
		sc.ClearLastUsedOutPoints()
		sc.SetLastUsedUtxoHeight(height)
		log.Info("Side chain [", sc.GetKey(), "] SetLastUsedUtxoHeight ", height)
	}
}

func (mc *MainChainImpl) CheckAndRemoveDepositTransactionsFromDB() error {
	//remove deposit transactions if exist on side chain
	txs, err := store.DbCache.MainChainStore.GetAllMainChainTxs()
	if err != nil {
		return err
	}

	if len(txs) == 0 {
		return nil
	}

	allSideChainTxHashes := make(map[arbitrator.SideChain][]string, 0)
	for _, tx := range txs {
		sc, ok := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(tx.GenesisBlockAddress)
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
		err = store.DbCache.MainChainStore.RemoveMainChainTxs(receivedTxs, finalGenesisAddresses)
		if err != nil {
			return err
		}
		err = store.FinishedTxsDbCache.AddSucceedDepositTxs(receivedTxs, finalGenesisAddresses)
		if err != nil {
			log.Error("[CheckAndRemoveDepositTransactionsFromDB] Add succeed deposit transactions into finished db failed")
		}
	}

	return nil
}

func InitMainChain(ar arbitrator.Arbitrator) error {
	currentArbitrator, ok := ar.(*arbitrator.ArbitratorImpl)
	if !ok {
		return errors.New("Unknown arbitrator type.")
	}

	mainChainServer := &MainChainImpl{&cs.DistributedNodeServer{}}
	cs.P2PClientSingleton.AddMainchainListener(mainChainServer)
	currentArbitrator.SetMainChain(mainChainServer)

	mainChainClient := &MainChainClientImpl{&cs.DistributedNodeClient{}}
	cs.P2PClientSingleton.AddMainchainListener(mainChainClient)
	currentArbitrator.SetMainChainClient(mainChainClient)

	return nil
}
