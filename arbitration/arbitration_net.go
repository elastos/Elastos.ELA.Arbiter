package arbitration

type ArbitrationNetListener interface {

	OnReceived(buf []byte, arbitrator Arbitrator)
}

type ArbitrationNet interface {

	Broadcast(buf []byte)

	AddListener(listener ArbitrationNetListener)
	RemoveListener(listener ArbitrationNetListener)
}