package mainchain

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA/common"

	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
)

type MainChainClientImpl struct {
	*cs.DistributedNodeClient
}

func (client *MainChainClientImpl) OnReceivedSignMsg(id peer.PID, content []byte) {
	if err := client.OnReceivedProposal(id, content); err != nil {
		log.Error("[OnReceivedSignMsg] mainchain client received distributed item message error: ", err)
	}
}

func (client *MainChainClientImpl) OnSendSchnorrItemMsg(id peer.PID, hash common.Uint256) {
	// no need to deal with this message
	return
}
