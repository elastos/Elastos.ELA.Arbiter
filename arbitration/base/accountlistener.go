package base

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type AccountListener interface {
	GetAccountAddress() string
	OnUTXOChanged(txinfos []*WithdrawTx, blockHeight uint32) error
	OnIllegalEvidenceFound(evidence *payload.SidechainIllegalData) error

	StartSideChainMining()
	SubmitAuxpow(genesishash string, blockhash string, submitauxpow string) error
	UpdateLastNotifySideMiningHeight(genesisBlockHash common.Uint256)
	UpdateLastSubmitAuxpowHeight(genesisBlockHash common.Uint256)

	SendCachedWithdrawTxs()
}

type AccountMonitor interface {
	AddListener(listener AccountListener)
	RemoveListener(account string) error

	SyncChainData()
}
