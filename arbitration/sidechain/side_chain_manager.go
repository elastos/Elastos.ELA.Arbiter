package sidechain

import (
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/common/config"
)

type SideChainManagerImpl struct {
	SideChains map[string]SideChain
}

func (sideManager *SideChainManagerImpl) AddChain(key string, chain SideChain) {
	sideManager.SideChains[key] = chain
}

func (sideManager *SideChainManagerImpl) GetChain(key string) (SideChain, bool) {
	elem, ok := sideManager.SideChains[key]
	return elem, ok
}

func (sideManager *SideChainManagerImpl) GetAllChains() []SideChain {
	var chains []SideChain
	for _, v := range sideManager.SideChains {
		chains = append(chains, v)
	}
	return chains
}

func init() {
	currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator().(*ArbitratorImpl)

	sideChainManager := &SideChainManagerImpl{SideChains: make(map[string]SideChain)}
	for _, sideConfig := range config.Parameters.SideNodeList {
		side := &SideChainImpl{
			Key:           sideConfig.GenesisBlockAddress,
			CurrentConfig: sideConfig,
		}

		sideChainManager.AddChain(sideConfig.GenesisBlockAddress, side)
	}
	currentArbitrator.SetSideChainManager(sideChainManager)
}
