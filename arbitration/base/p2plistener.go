package base

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
)

type MainchainMsgListener interface {
	OnReceivedSignMsg(id peer.PID, content []byte)
	OnSendSchnorrItemMsg(id peer.PID, hash common.Uint256)
}
