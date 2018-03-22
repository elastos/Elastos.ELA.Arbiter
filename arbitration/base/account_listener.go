package base

type AccountListener interface {
	GetAccountAddress() string
	OnUTXOChanged(txinfo *TransactionInfo) error
}

type AccountMonitor interface {
	AddListener(listener AccountListener)
	RemoveListener(account string) error

	SyncChainData()
}
