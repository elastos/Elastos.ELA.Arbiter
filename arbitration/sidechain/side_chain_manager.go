package sidechain

import (
	. "Elastos.ELA.Arbiter/arbitration/arbitrator"
	"Elastos.ELA.Arbiter/common/config"
)

type SideChainManagerImpl struct {
	sideChains map[string]SideChain
}

func (sideManager *SideChainManagerImpl) AddChain(key string, chain SideChain) {
	sideManager.sideChains[key] = chain
}

func (sideManager *SideChainManagerImpl) GetChain(key string) (SideChain, bool) {
	elem, ok := sideManager.sideChains[key]
	return elem, ok
}

func (sideManager *SideChainManagerImpl) GetAllChains() []SideChain {
	var chains []SideChain
	for _, v := range sideManager.sideChains {
		chains = append(chains, v)
	}
	return chains
}

func init() {
	currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator().(*ArbitratorImpl)

	sideChainManager := &SideChainManagerImpl{sideChains: make(map[string]SideChain)}
	for _, sideConfig := range config.Parameters.SideNodeList {
		side := &SideChainImpl{
			key:           sideConfig.GenesisBlockAddress,
			currentConfig: sideConfig,
		}

		sideChainManager.AddChain(sideConfig.GenesisBlockAddress, side)
	}
	currentArbitrator.SetSideChainManager(sideChainManager)
}
