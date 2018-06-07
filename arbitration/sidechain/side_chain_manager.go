package sidechain

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
)

type SideChainManagerImpl struct {
	SideChains map[string]arbitrator.SideChain
}

func (sideManager *SideChainManagerImpl) AddChain(key string, chain arbitrator.SideChain) {
	sideManager.SideChains[key] = chain
}

func (sideManager *SideChainManagerImpl) GetChain(key string) (arbitrator.SideChain, bool) {
	elem, ok := sideManager.SideChains[key]
	return elem, ok
}

func (sideManager *SideChainManagerImpl) GetAllChains() []arbitrator.SideChain {
	var chains []arbitrator.SideChain
	for _, v := range sideManager.SideChains {
		chains = append(chains, v)
	}
	return chains
}

func (sideManager *SideChainManagerImpl) StartSideChainMining() {
	for _, sc := range sideManager.SideChains {
		log.Info("[OnDutyChanged] Start side chain mining: genesis address [", sc.GetKey(), "]")
		sc.StartSideChainMining()
	}
}

func Init() {
	currentArbitrator, ok := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().(*arbitrator.ArbitratorImpl)
	if !ok {
		return
	}

	sideChainManager := &SideChainManagerImpl{SideChains: make(map[string]arbitrator.SideChain)}
	for _, sideConfig := range config.Parameters.SideNodeList {
		side := &SideChainImpl{
			Key:           sideConfig.GenesisBlockAddress,
			CurrentConfig: sideConfig,
		}

		sideChainManager.AddChain(sideConfig.GenesisBlockAddress, side)
	}
	currentArbitrator.SetSideChainManager(sideChainManager)
}
