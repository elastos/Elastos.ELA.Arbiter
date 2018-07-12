package arbitrator

import (
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	. "github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/bloom"
	. "github.com/elastos/Elastos.ELA/core"
	ela "github.com/elastos/Elastos.ELA/core"
)

type DepositListener struct {
	ListenAddress string
	notifyQueue   chan *notifyTask
}

func (l *DepositListener) Address() string {
	return l.ListenAddress
}

func (l *DepositListener) Type() TransactionType {
	return TransferCrossChainAsset
}

func (l *DepositListener) Flags() uint64 {
	return FlagNotifyConfirmed | FlagNotifyInSyncing
}

func (l *DepositListener) Notify(id common.Uint256, proof bloom.MerkleProof, tx ela.Transaction) {
	log.Info("[Notify-Deposit] find deposit transaction and add into channel, transaction hash:", tx.Hash().String())
	l.notifyQueue <- &notifyTask{id, &proof, &tx}
}

func (l *DepositListener) ProcessNotifyData(tasks []*notifyTask) {
	log.Info("[Notify-Process] deal with", len(tasks), "transactions")

	var ids []common.Uint256
	var txs []*ela.Transaction
	var proofs []*bloom.MerkleProof
	var txHashes []string
	var genesisAddresses []string
	for _, data := range tasks {
		ids = append(ids, data.id)
		txs = append(txs, data.tx)
		proofs = append(proofs, data.proof)
		txHashes = append(txHashes, data.tx.Hash().String())
		genesisAddresses = append(genesisAddresses, l.ListenAddress)
	}

	result, err := store.DbCache.MainChainStore.AddMainChainTxs(txHashes, genesisAddresses, txs, proofs)
	if err != nil {
		log.Error("AddMainChainTx error:", err)
		return
	}

	for i := 0; i < len(ids); i++ {
		spvService.SubmitTransactionReceipt(ids[i], txs[i].Hash())
	}

	if !ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfMain() {
		return
	}

	var finalTxs []*ela.Transaction
	var finalProofs []*bloom.MerkleProof
	for i := 0; i < len(result); i++ {
		if result[i] {
			finalTxs = append(finalTxs, txs[i])
			finalProofs = append(finalProofs, proofs[i])
		}
	}
	log.Info("[Notify-Process] find deposit transaction, create and send deposit transaction, size of txs:", len(finalTxs))
	for index, tx := range finalTxs {
		log.Info("[Notify-Process] tx hash[", index, "]:", tx.Hash().String())
	}
	ArbitratorGroupSingleton.GetCurrentArbitrator().CreateAndSendDepositTransactions(finalProofs, finalTxs, l.ListenAddress)
}

func (l *DepositListener) Rollback(height uint32) {
}

type notifyTask struct {
	id    common.Uint256
	proof *bloom.MerkleProof
	tx    *ela.Transaction
}

func (l *DepositListener) start() {
	l.notifyQueue = make(chan *notifyTask, 10000)
	go func() {
		var tasks []*notifyTask
		for {
			select {
			case data, ok := <-l.notifyQueue:
				if ok {
					tasks = append(tasks, data)
					log.Info("[DepositListener] len tasks:", len(tasks))
					if len(tasks) >= 10000 {
						l.ProcessNotifyData(tasks)
						tasks = make([]*notifyTask, 0)
					}
				}
			default:
				if len(tasks) > 0 {
					l.ProcessNotifyData(tasks)
					tasks = make([]*notifyTask, 0)
				}
				data, ok := <-l.notifyQueue
				if ok {
					tasks = append(tasks, data)
					log.Info("[DepositListener] len tasks:", len(tasks))
				}
			}
		}
	}()
}
