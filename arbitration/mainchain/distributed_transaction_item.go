package mainchain

import (
	"Elastos.ELA.Arbiter/arbitration/arbitrator"
	tx "Elastos.ELA.Arbiter/core/transaction"
)

type DistributedTransactionItem struct {
	rawTransaction *tx.Transaction
}

func (item *DistributedTransactionItem) InitScript(arbitrator arbitrator.Arbitrator) error {
	return nil
}

func (item *DistributedTransactionItem) Serialize() ([]byte, error) {
	return nil, nil
}

func (item *DistributedTransactionItem) Deserialize(content []byte) error {
	return nil
}
