package sidechain

import (
	"errors"
	"sync"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

const sideChainHeightInterval uint32 = 1000

type SideChainAccountMonitorImpl struct {
	mux sync.Mutex

	ParentArbitrator   arbitrator.Arbitrator
	accountListenerMap map[string]base.AccountListener
}

func (monitor *SideChainAccountMonitorImpl) tryInit() {
	if monitor.accountListenerMap == nil {
		monitor.accountListenerMap = make(map[string]base.AccountListener)
	}
}

func (monitor *SideChainAccountMonitorImpl) AddListener(listener base.AccountListener) {
	monitor.tryInit()
	monitor.accountListenerMap[listener.GetAccountAddress()] = listener
}

func (monitor *SideChainAccountMonitorImpl) RemoveListener(account string) error {
	if monitor.accountListenerMap == nil {
		return nil
	}

	if _, ok := monitor.accountListenerMap[account]; !ok {
		return errors.New("do not exist listener")
	}
	delete(monitor.accountListenerMap, account)
	return nil
}

func (monitor *SideChainAccountMonitorImpl) fireUTXOChanged(withdrawTxs []*base.WithdrawTx, genesisBlockAddress string, blockHeight uint32) error {
	if monitor.accountListenerMap == nil {
		return nil
	}

	item, ok := monitor.accountListenerMap[genesisBlockAddress]
	if !ok {
		return errors.New("fired unknown listener")
	}

	return item.OnUTXOChanged(withdrawTxs, blockHeight)
}

func (monitor *SideChainAccountMonitorImpl) fireIllegalEvidenceFound(evidence *payload.SidechainIllegalData) error {
	if monitor.accountListenerMap == nil {
		return nil
	}

	item, ok := monitor.accountListenerMap[evidence.GenesisBlockAddress]
	if !ok {
		return errors.New("fired unknown listener")
	}

	return item.OnIllegalEvidenceFound(evidence)
}

func (monitor *SideChainAccountMonitorImpl) SyncChainData(sideNode *config.SideNodeConfig) {
	for {
		chainHeight, currentHeight, needSync := monitor.needSyncBlocks(sideNode.GenesisBlockAddress, sideNode.Rpc)

		if needSync {
			if currentHeight < sideNode.SyncStartHeight {
				currentHeight = sideNode.SyncStartHeight
			}
			log.Info("[SyncSideChain] side chain:", sideNode.GenesisBlockAddress,
				"current height:", currentHeight, " chain height:", chainHeight)
			count := uint32(1)
			for currentHeight < chainHeight {
				if currentHeight >= 6 {
					transactions, err := rpc.GetWithdrawTransactionByHeight(currentHeight+1-6, sideNode.Rpc)
					if err != nil {
						log.Error("get destroyed transaction at height:", currentHeight+1-6, "failed\n"+
							"rpc:", sideNode.Rpc.IpAddress, ":", sideNode.Rpc.HttpJsonPort, "\n"+
							"error:", err)
						break
					}
					monitor.processTransactions(transactions, sideNode.GenesisBlockAddress, currentHeight+1-6)
				}

				evidences, err := rpc.GetIllegalEvidenceByHeight(currentHeight+1, sideNode.Rpc)
				if err != nil {
					log.Error("get illegal evidence at height:", currentHeight+1, "failed\n"+
						"rpc:", sideNode.Rpc.IpAddress, ":", sideNode.Rpc.HttpJsonPort, "\n"+
						"error:", err)
					break
				}

				// process illegal evidences
				for _, e := range evidences {
					se, err := common.Uint256FromHexString(e.Evidence)
					if err != nil {
						log.Error("invalid evidence:", err.Error())
						continue
					}
					sce, err := common.Uint256FromHexString(e.CompareEvidence)
					if err != nil {
						log.Error("invalid evidence:", err.Error())
						continue
					}
					illegalSigner, err := common.HexStringToBytes(e.IllegalSigner)
					if err != nil {
						log.Error("invalid illegal signer:", err.Error())
						continue
					}

					evidence := &payload.SidechainIllegalData{
						IllegalType:         payload.IllegalDataType(e.IllegalType),
						Height:              currentHeight + 1,
						IllegalSigner:       illegalSigner,
						Evidence:            payload.SidechainIllegalEvidence{*se},
						CompareEvidence:     payload.SidechainIllegalEvidence{*sce},
						GenesisBlockAddress: sideNode.GenesisBlockAddress,
					}
					if se.String() > sce.String() {
						evidence.Evidence =
							payload.SidechainIllegalEvidence{*sce}
						evidence.CompareEvidence =
							payload.SidechainIllegalEvidence{*se}
					}

					if err := monitor.fireIllegalEvidenceFound(
						evidence); err != nil {
						log.Error("fire illegal evidence found error:",
							err.Error())
					}
				}
				currentHeight++
				count++
				if count%sideChainHeightInterval == 0 {
					currentHeight = store.DbCache.SideChainStore.CurrentSideHeight(sideNode.GenesisBlockAddress, currentHeight)
					log.Info(" [SyncSideChain] Side chain [", sideNode.GenesisBlockAddress, "] height: ", currentHeight)
				}
			}
			// Update wallet height
			currentHeight = store.DbCache.SideChainStore.CurrentSideHeight(sideNode.GenesisBlockAddress, currentHeight)
			log.Info(" [SyncSideChain] Side chain [", sideNode.GenesisBlockAddress, "] height: ", currentHeight)

			if arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfMain() {
				sideChain, ok := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(sideNode.GenesisBlockAddress)
				if ok {
					sideChain.StartSideChainMining()
					log.Info("[SyncSideChain] Start side chain mining, genesis address: [", sideNode.GenesisBlockAddress, "]")
				}
			}

		}

		time.Sleep(time.Millisecond * config.Parameters.SideChainMonitorScanInterval)
	}
}

func (monitor *SideChainAccountMonitorImpl) needSyncBlocks(genesisBlockAddress string, config *config.RpcConfig) (uint32, uint32, bool) {

	chainHeight, err := rpc.GetCurrentHeight(config)
	if err != nil {
		return 0, 0, false
	}

	currentHeight := store.DbCache.SideChainStore.CurrentSideHeight(genesisBlockAddress, store.QueryHeightCode)

	if currentHeight >= chainHeight {
		return chainHeight, currentHeight, false
	}

	return chainHeight, currentHeight, true
}

func (monitor *SideChainAccountMonitorImpl) processTransactions(transactions []*base.WithdrawTxInfo, genesisAddress string, blockHeight uint32) {
	var withdrawTxs []*base.WithdrawTx
	for _, txn := range transactions {
		txnBytes, err := common.HexStringToBytes(txn.TxID)
		if err != nil {
			log.Warn("Find output to destroy address, but transaction hash to transaction bytes failed")
			continue
		}
		reversedTxnBytes := common.BytesReverse(txnBytes)
		hash, err := common.Uint256FromBytes(reversedTxnBytes)
		if err != nil {
			log.Warn("Find output to destroy address, but reversed transaction hash bytes to transaction hash failed")
			continue
		}

		var withdrawAssets []*base.WithdrawAsset
		for _, withdraw := range txn.CrossChainAssets {
			opAmount, err := common.StringToFixed64(withdraw.OutputAmount)
			if err != nil {
				log.Warn("Find output to destroy address, but have invlaid corss chain output amount")
				continue
			}
			csAmount, err := common.StringToFixed64(withdraw.CrossChainAmount)
			if err != nil {
				log.Warn("Find output to destroy address, but have invlaid corss chain amount")
				continue
			}
			programHash, err := common.Uint168FromAddress(withdraw.CrossChainAddress)
			if err != nil {
				log.Warn("invalid withdraw cross chain address:", withdraw.CrossChainAddress)
				continue
			}
			addr, err := programHash.ToAddress()
			if err != nil || addr != withdraw.CrossChainAddress {
				log.Warn("invalid withdraw cross chain address:", withdraw.CrossChainAddress)
				continue
			}
			if contract.PrefixType(programHash[0]) != contract.PrefixStandard &&
				contract.PrefixType(programHash[0]) != contract.PrefixMultiSig {
				log.Warn("invalid withdraw cross chain address:", withdraw.CrossChainAddress)
				continue
			}

			withdrawAssets = append(withdrawAssets, &base.WithdrawAsset{
				TargetAddress:    withdraw.CrossChainAddress,
				Amount:           opAmount,
				CrossChainAmount: csAmount,
			})
		}

		if len(withdrawAssets) == 0 {
			continue
		}

		withdrawTx := &base.WithdrawTx{
			Txid: hash,
			WithdrawInfo: &base.WithdrawInfo{
				WithdrawAssets: withdrawAssets,
			},
		}

		reversedTxnHash := common.BytesToHexString(reversedTxnBytes)
		if ok, err := store.DbCache.SideChainStore.HasSideChainTx(reversedTxnHash); err != nil || !ok {
			withdrawTxs = append(withdrawTxs, withdrawTx)
		}
	}
	if len(withdrawTxs) != 0 {
		err := monitor.fireUTXOChanged(withdrawTxs, genesisAddress, blockHeight)
		if err != nil {
			log.Error("[fireUTXOChanged] err:", err.Error())
		}
	}
}
