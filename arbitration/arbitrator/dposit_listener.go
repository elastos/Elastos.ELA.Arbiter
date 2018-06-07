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
	if ok, _ := store.DbCache.HasMainChainTx(tx.Hash().String()); ok {
		return
	}

	if err := store.DbCache.AddMainChainTx(tx.Hash().String(), &tx, &proof); err != nil {
		log.Error("AddMainChainTx error, txHash:", tx.Hash().String())
		return
	}

	spvService.SubmitTransactionReceipt(id, tx.Hash())

	if !ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfMain() {
		return
	}

	log.Info("[Notify-Deposit] find deposit transaction and createAndSendDepositTransaction")
	ArbitratorGroupSingleton.GetCurrentArbitrator().CreateAndSendDepositTransaction(&proof, &tx)
}

func (l *DepositListener) Rollback(height uint32) {
}
