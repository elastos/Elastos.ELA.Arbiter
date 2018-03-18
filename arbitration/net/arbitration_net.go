package net

type ArbitrationNetListener interface {

	OnReceived(buf []byte, arbitratorIndex int)
}

type ArbitrationNet interface {

	Broadcast(buf []byte)

	AddListener(listener ArbitrationNetListener)
	RemoveListener(listener ArbitrationNetListener)
}