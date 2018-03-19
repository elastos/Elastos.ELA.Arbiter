package arbitratorgroup

import (
	"fmt"
	"errors"
	"Elastos.ELA.Arbiter/common/config"
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
}

type ArbitratorGroupImpl struct {

	arbitrators []Arbitrator
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

func init() {
	ArbitratorGroupSingleton = ArbitratorGroupImpl{}
	fmt.Println("member count: ", config.Parameters.MemberCount)

	foundation := new(ArbitratorImpl)
	ArbitratorGroupSingleton.arbitrators = append(ArbitratorGroupSingleton.arbitrators, foundation)
	ArbitratorGroupSingleton.currentArbitrator = 0
}