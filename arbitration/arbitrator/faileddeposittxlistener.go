package arbitrator

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
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

					// 这个depositTx需要组合，1个交易可能有多个depositAsset对应多条链转账，去除掉不是支持SupportQuickRecharge的但是
					// 因为我们本身就是从支持quickrecharge的链查询的，所以应该没有问题，然后就是合并，多个交易可能对应的target地址是一样的，如果是这样就需要
					// 合并为同一个交易发送。
					var failedTxs []base.FailedDepositTx
					if err := rpc.Unmarshal(&resp, &failedTxs); err != nil {
						log.Error("[MoniterFailedDepositTransfer] Unmarshal getfaileddeposittransactions responce error")
						break
					}

					if !ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfMain() {
						log.Warn("[MoniterFailedDepositTransfer] i am not onduty")
						return
					}

					err = curr.SendFailedDepositTxs(failedTxs)
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
