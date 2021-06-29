package arbitrator

import (
	"bytes"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/common"
)

const MinCrossChainTxFee common.Fixed64 = 10000

func MonitorInvalidWithdrawTransaction() {
	for {
		select {
		case <-time.After(time.Second * 5):
			mainChainHeight := store.DbCache.MainChainStore.CurrentHeight(store.QueryHeightCode)
			if mainChainHeight < config.Parameters.ProcessInvalidWithdrawHeight {
				continue
			}

			currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()
			ar := ArbitratorGroupSingleton.listener.(*ArbitratorImpl)
			for _, sc := range ar.sideChainManagerImpl.GetAllChains() {
				if !sc.GetCurrentConfig().SupportInvalidWithdraw {
					continue
				}
				txHashes, _, err := store.DbCache.SideChainStore.GetAllSideChainTxHashesAndHeights(sc.GetKey())
				if err != nil {
					continue
				}

				if len(txHashes) == 0 {
					continue
				}

				unsolvedTransactions, err := store.DbCache.SideChainStore.GetSideChainTxsFromHashes(txHashes)
				if err != nil {
					continue
				}

				if len(unsolvedTransactions) == 0 {
					continue
				}

				log.Info("Found unsolvedTransactions count:", len(unsolvedTransactions))
				// get all invalid transactions
				invalidTransactions := make([]*base.WithdrawTx, 0)
				for _, tx := range unsolvedTransactions {
					ignore := false
					for _, w := range tx.WithdrawInfo.WithdrawAssets {
						if *w.CrossChainAmount <= 0 ||
							*w.Amount-*w.CrossChainAmount < MinCrossChainTxFee {
							ignore = true
							break
						}
						_, err := common.Uint168FromAddress(w.TargetAddress)
						if err != nil {
							ignore = true
							break
						}
					}
					if ignore {
						invalidTransactions = append(invalidTransactions, tx)
						continue
					}

					if len(tx.WithdrawInfo.WithdrawAssets) == 0 {
						invalidTransactions = append(invalidTransactions, tx)
					}
				}

				// get all not processed invalid withdraw transactions
				allHashes := make([]string, 0)
				for _, tx := range invalidTransactions {
					allHashes = append(allHashes, common.ToReversedString(*tx.Txid))
				}
				processedTxs, err := sc.GetProcessedInvalidWithdrawTransactions(allHashes)
				if err != nil {
					log.Error("[GetProcessedInvalidWithdrawTransactions] Error:", err)
					continue
				}
				log.Info("[GetProcessedInvalidWithdrawTransactions] processedTxs:", processedTxs)

				// remove already processed invalid withdraw transactions

				reversedProcessedTxs := make([]string, 0)
				for _, t := range processedTxs {
					bytes, err := common.FromReversedString(t)
					if err != nil {
						log.Error("invalid processed tx:", t)
						continue
					}
					reversedProcessedTxs = append(reversedProcessedTxs, common.BytesToHexString(bytes))
				}

				err = store.DbCache.SideChainStore.RemoveSideChainTxs(reversedProcessedTxs)
				if err != nil {
					log.Error("failed to remove failed withdraw transaction from db")
				}

				// get already processed transactions map
				processedTxsMap := make(map[string]struct{}, 0)
				for _, ptx := range processedTxs {
					processedTxsMap[ptx] = struct{}{}
				}

				// broadcast not processed invalid withdraw transaction
				for _, tx := range invalidTransactions {
					txHash := tx.Txid.String()

					// filter already processed txs
					if _, ok := processedTxsMap[txHash]; ok {
						continue
					}

					// sign transaction hash
					buf := new(bytes.Buffer)
					withdrawTxHash, err := common.Uint256FromHexString(common.ToReversedString(*tx.Txid))
					if err := withdrawTxHash.Serialize(buf); err != nil {
						log.Error("failed to serialize invalid transaction hash")
						continue
					}
					signature, err := currentArbitrator.Sign(buf.Bytes())
					if err != nil {
						log.Error("failed to sign invalid transaction hash")
						continue
					}

					// send transaction to side chain.
					_, err = sc.SendInvalidWithdrawTransaction(signature, common.ToReversedString(*tx.Txid))
					if err != nil {
						log.Error("[SendInvalidWithdrawTransactions] Error", err.Error())
					} else {
						log.Info("[SendInvalidWithdrawTransactions] transactions hash: ", txHash)
					}
				}
			}
		}
	}

}
