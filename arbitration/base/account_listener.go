package base

type AccountListener interface {
	GetAccountAddress() string
	OnUTXOChanged(txinfo *TransactionInfo) error
	OnDutyArbitratorChanged(onDuty bool)
}

type AccountMonitor interface {
	AddListener(listener AccountListener)
	RemoveListener(account string) error

	SyncChainData()
}
