package base

import (
	peer2 "github.com/elastos/Elastos.ELA/dpos/p2p/peer"
)

type MainchainMsgListener interface {
	OnReceivedSignMsg(id peer2.PID, content []byte)
}
