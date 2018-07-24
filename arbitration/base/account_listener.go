package base

type AccountListener interface {
	GetAccountAddress() string
	OnUTXOChanged(txinfos []*TransactionInfo, blockHeight uint32) error

	StartSideChainMining()
	SubmitAuxpow(genesishash string, blockhash string, submitauxpow string) error
	UpdateLastNotifySideMiningHeight(addr string)
	UpdateLastSubmitAuxpowHeight(addr string)

	SendCachedWithdrawTxs() error
}

type AccountMonitor interface {
	AddListener(listener AccountListener)
	RemoveListener(account string) error

	SyncChainData()
}
