package arbitration

type SideChainNode interface {
	GetCurrentHeight() (uint32, error)
	GetBlockByHeight(height uint32) (BlockInfo)

	SendTransaction(info *TransactionInfo) error
}
