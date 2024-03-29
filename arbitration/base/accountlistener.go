package base

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type AccountListener interface {
	GetAccountAddress() string
	OnUTXOChanged(withdrawTxs []*WithdrawTx, blockHeight uint32) error
	OnIllegalEvidenceFound(evidence *payload.SidechainIllegalData) error
	OnNFTChanged(nftDestroyTxs []* NFTDestroyFromSideChainTx, blockHeight uint32) error

	StartSideChainMining()
	SubmitAuxpow(genesishash string, blockhash string, submitauxpow string) error
	UpdateLastNotifySideMiningHeight(genesisBlockHash common.Uint256)
	UpdateLastSubmitAuxpowHeight(genesisBlockHash common.Uint256)

	SendCachedWithdrawTxs(currentHeight uint32)
	SendCachedNFTDestroyTxs(currentHeight uint32)
	SendCachedReturnDepositTxs()
	SendFailedDepositTxs(failedTxs []*FailedDepositTx) error
}

type AccountMonitor interface {
	AddListener(listener AccountListener)
	RemoveListener(account string) error

	SyncChainData()
}
