package arbitration

type ArbitratorsElection interface {

}

type ArbitratorGroup interface {
	ArbitratorsElection

	init() error
	GetCurrentArbitrator() Arbitrator
	GetArbitratorsCount() uint32
}