package arbitrator

import (
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA/common"
	"time"
)

func MoniterFailedDepositTransfer() {
	for {
		select {
		case <-time.After(time.Second * 1):

			currentArbitrator, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().(*ArbitratorImpl)
			if !ok {
				log.Error("[MoniterFailedDepositTransfer] Unable to get current arbiter")
				break
			}

			for _, curr := range currentArbitrator.GetAllChains() {
				cfg := curr.GetCurrentConfig()
				if cfg.SupportQuickRecharge {

					resp, err := rpc.CallAndUnmarshal("getfaileddeposittransactions", nil,
						cfg.Rpc)
					if err != nil {
						log.Errorf("[MoniterFailedDepositTransfer] Unable to call getfaileddeposittransactions rpc ")
						break
					}

					var failedTxs []string
					if err := rpc.Unmarshal(&resp, &failedTxs); err != nil {
						log.Error("[MoniterFailedDepositTransfer] Unmarshal getfaileddeposittransactions responce error")
						break
					}

					if !ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfMain() {
						log.Warn("[MoniterFailedDepositTransfer] i am not onduty")
						return
					}

					failedTxsUint256, err := ToUint256(failedTxs)
					if err != nil {
						log.Errorf("[MoniterFailedDepositTransfer] Unable to call ToUint256 ")
						break
					}

					err = curr.SendFailedDepositTxs(failedTxsUint256, cfg.GenesisBlockAddress)
					if err != nil {
						log.Error("[MoniterFailedDepositTransfer] CreateAndBroadcastWithdrawProposal failed")
						break
					}
				}
			}
		}
	}

}

func ToUint256(failedTxs []string) ([]common.Uint256, error) {
	var ret []common.Uint256
	for _, tx := range failedTxs {
		data, err := common.Uint256FromHexString(tx)
		if err != nil {
			return nil, err
		}
		ret = append(ret, *data)
	}
	return ret, nil
}
