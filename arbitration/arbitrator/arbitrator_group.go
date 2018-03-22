package arbitrator

import (
	"Elastos.ELA.Arbiter/common/config"
	"SPVWallet/interface"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

var (
	ArbitratorGroupSingleton ArbitratorGroupImpl
)

type ArbitratorsElection interface {
}

type ArbitratorGroup interface {
	ArbitratorsElection

	GetCurrentArbitrator() (Arbitrator, error)
	GetArbitratorsCount() int
	GetAllArbitrators() []string
}

type ArbitratorGroupImpl struct {
	arbitrators       []Arbitrator
	currentArbitrator int
}

func (group *ArbitratorGroupImpl) GetArbitratorsCount() int {
	return len(group.arbitrators)
}

func (group *ArbitratorGroupImpl) GetCurrentArbitrator() (Arbitrator, error) {
	if group.currentArbitrator >= group.GetArbitratorsCount() {
		return nil, errors.New("Can not find current arbitrator!")
	}
	return group.arbitrators[group.currentArbitrator], nil
}

func (group *ArbitratorGroupImpl) GetAllArbitrators() []string {
	return nil
}

func init() {
	ArbitratorGroupSingleton = ArbitratorGroupImpl{}
	fmt.Println("member count: ", config.Parameters.MemberCount)

	foundation := new(ArbitratorImpl)
	ArbitratorGroupSingleton.arbitrators = append(ArbitratorGroupSingleton.arbitrators, foundation)
	ArbitratorGroupSingleton.currentArbitrator = 0

	// SPV module init
	var err error
	publicKey := foundation.GetPublicKey()
	publicKeyBytes, _ := publicKey.EncodePoint(true)
	foundation.spvService, err = _interface.NewSPVService(binary.LittleEndian.Uint64(publicKeyBytes))
	if err != nil {
		fmt.Println("[Error] " + err.Error())
		os.Exit(1)
	}
	for _, sideNode := range config.Parameters.SideNodeList {
		foundation.spvService.RegisterAccount(sideNode.GenesisBlockAddress)
	}
	foundation.spvService.OnTransactionConfirmed(foundation.OnTransactionConfirmed)
	foundation.spvService.Start()
}
