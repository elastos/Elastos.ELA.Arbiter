package sideauxpow

import (
	"errors"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
)

func SubmitAuxpow(genesishash string, blockhash string, submitauxpow string) error {
	log.Info("submitauxblock")

	var sideNode *config.SideNodeConfig
	for _, node := range config.Parameters.SideNodeList {
		if node.GenesisBlock == genesishash {
			sideNode = node
		}
	}
	if sideNode == nil {
		return errors.New("invalid side node")
	}

	params := make(map[string]string, 2)
	params["blockhash"] = blockhash
	params["sideauxpow"] = submitauxpow
	resp, err := rpc.CallAndUnmarshal("submitauxblock", params, sideNode.Rpc)
	if err != nil {
		return err
	}

	log.Info(resp)
	return nil
}
