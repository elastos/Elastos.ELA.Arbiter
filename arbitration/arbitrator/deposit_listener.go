package arbitrator

import (
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA.SPV/bloom"
	. "github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA/common"
	. "github.com/elastos/Elastos.ELA/core/types"
	ela "github.com/elastos/Elastos.ELA/core/types"
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
	var txs []*MainChainTransaction
	for _, data := range tasks {
		ids = append(ids, data.id)
		txs = append(txs, &MainChainTransaction{
			TransactionHash:     data.tx.Hash().String(),
			GenesisBlockAddress: l.ListenAddress,
			Transaction:         data.tx,
			Proof:               data.proof,
		})
	}

	result, err := store.DbCache.MainChainStore.AddMainChainTxs(txs)
	if err != nil {
		log.Error("[Notify-Process] AddMainChainTx error:", err)
		return
	}

	for i := 0; i < len(ids); i++ {
		SpvService.SubmitTransactionReceipt(ids[i], txs[i].Transaction.Hash())
	}

	if !ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfMain() {
		log.Warn("[Notify-Process] i am not onduty")
		return
	}

	var spvTxs []*SpvTransaction
	for i := 0; i < len(result); i++ {
		if result[i] {
			spvTxs = append(spvTxs, &SpvTransaction{MainChainTransaction: txs[i].Transaction, Proof: txs[i].Proof})
		}
	}
	log.Info("[Notify-Process] find deposit transaction, create and send deposit transaction, size of txs:", len(spvTxs))
	for index, spvTx := range spvTxs {
		log.Info("[Notify-Process] tx hash[", index, "]:", spvTx.MainChainTransaction.Hash().String())
	}
	ArbitratorGroupSingleton.GetCurrentArbitrator().SendDepositTransactions(spvTxs, l.ListenAddress)
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
