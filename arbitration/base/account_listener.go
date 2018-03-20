package base

import (
	. "Elastos.ELA.Arbiter/common"
)

type AccountListener interface {
	GetAccountAddress() string
	OnUTXOChanged(transactionHash Uint256) error
}

type AccountMonitor interface {
	AddListener(listener AccountListener)
	RemoveListener(account string) error

	SyncChainData()
}
