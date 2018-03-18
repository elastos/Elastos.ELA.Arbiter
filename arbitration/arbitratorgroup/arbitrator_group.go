package arbitratorgroup

import (
	"Elastos.ELA.Arbiter/arbitration/base"
	"fmt"
	"errors"
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
	fmt.Println("member count: ", base.Parameters.MemberCount)

	fundation := new(ArbitratorImpl)
	ArbitratorGroupSingleton.arbitrators = append(ArbitratorGroupSingleton.arbitrators, fundation)
	ArbitratorGroupSingleton.currentArbitrator = 0
}