package base

type AccountListener interface {
	GetAccountAddress() string
	OnUTXOChanged(txinfos []*TransactionInfo, blockHeight uint32) error
	StartSidechainMining()
	SyncSideChainCachedTxs() error
}

type AccountMonitor interface {
	AddListener(listener AccountListener)
	RemoveListener(account string) error

	SyncChainData()
}
