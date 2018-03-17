package arbitration

type ArbitratorMain interface {
	MainChain
}

type ArbitratorSide interface {
	SideChainManager
}

type Arbitrator interface {
	ArbitratorMain
	ArbitratorSide
	ArbitrationNetListener
	ComplainListener

	GetArbitrationNet() ArbitrationNet
	GetComplainSolving() ComplainSolving

	IsOnDuty() bool
	GetArbitratorGroup() ArbitratorGroup
}