package base

import (
	. "Elastos.ELA.Arbiter/common"
)

type AccountListener interface {
	OnUTXOChanged(transactionHash Uint256) error
}

type AccountMonitor interface {
	SetAccount(account string) error

	AddListener(listener AccountListener) error
	RemoveListener(listener AccountListener) error

	fireUTXOChanged() error
}

type AccountMonitorImpl struct {
}
