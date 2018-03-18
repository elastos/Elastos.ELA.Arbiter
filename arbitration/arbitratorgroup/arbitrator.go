package arbitratorgroup

import (
	"Elastos.ELA.Arbiter/arbitration/net"
	main "Elastos.ELA.Arbiter/arbitration/mainchain"
	side "Elastos.ELA.Arbiter/arbitration/sidechain"
	comp "Elastos.ELA.Arbiter/arbitration/complain"
)

type ArbitratorMain interface {
	main.MainChain
}

type ArbitratorSide interface {
	side.SideChainManager
}

type Arbitrator interface {
	ArbitratorMain
	ArbitratorSide
	net.ArbitrationNetListener
	comp.ComplainListener

	GetArbitrationNet() net.ArbitrationNet
	GetComplainSolving() comp.ComplainSolving

	IsOnDuty() bool
	GetArbitratorGroup() ArbitratorGroup
}