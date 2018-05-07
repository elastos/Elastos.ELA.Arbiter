package sideauxpow

import (
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
)

func SubmitAuxpow(blockhash string, submitauxpow string) error {
	log.Info("submitauxblock")
	params := make(map[string]string, 2)
	params["blockhash"] = blockhash
	params["sideauxpow"] = submitauxpow
	resp, err := rpc.CallAndUnmarshal("submitauxblock", params, config.Parameters.SideNodeList[0].Rpc)
	if err != nil {
		return err
	}

	log.Info(resp)
	return nil
}
