package sideauxpow

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	i "github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA.SPV/log"
)

var spv i.SPVService

func StartSPV() {
	log.Init()

	var id = make([]byte, 8)
	var clientId uint64
	var err error
	rand.Read(id)
	binary.Read(bytes.NewReader(id), binary.LittleEndian, clientId)
	spv = i.NewSPVService(clientId, config.Parameters.SeedList)

	// Register account
	err = spv.RegisterAccount("EJsnmbFTyhhaVcGFszdZ49dKgd9BzgkZih")
	if err != nil {
		log.Error("Register account error: ", err)
		os.Exit(0)
	}

	// Set on transaction confirmed callback
	// spv.RegisterTransactionListener(&UnconfirmedListener{txType: ela.SideMining})

	// Start spv service
	spv.Start()
}

func SubmitAuxpow(blockhash string, submitauxpow string) error {
	fmt.Println("submitauxblock")
	params := make(map[string]string, 2)
	params["blockhash"] = blockhash
	params["sideauxpow"] = submitauxpow
	resp, err := rpc.CallAndUnmarshal("submitauxblock", params, config.Parameters.SideNodeList[0].Rpc)
	if err != nil {
		return err
	}

	fmt.Println(resp)
	return nil
}
