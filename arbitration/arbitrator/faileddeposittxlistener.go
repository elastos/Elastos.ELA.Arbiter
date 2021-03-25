package arbitrator

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA/common"
	"time"
)

func MoniterFailedDepositTransfer() {
	for {
		select {
		case <-time.After(time.Second * 30):
			log.Info("Start Monitor Failed Deposit Transfer")
			currentArbitrator, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().(*ArbitratorImpl)
			if !ok {
				log.Error("[MoniterFailedDepositTransfer] Unable to get current arbiter")
				break
			}

			for _, curr := range currentArbitrator.GetAllChains() {
				cfg := curr.GetCurrentConfig()
				if cfg.SupportQuickRecharge {

					//resp, err := rpc.CallAndUnmarshal("getfaileddeposittransactions", nil,
					//	cfg.Rpc)
					//if err != nil {
					//	log.Errorf("[MoniterFailedDepositTransfer] Unable to call getfaileddeposittransactions rpc ")
					//	break
					//}

					var failedTxs []base.FailedDepositTx
					//if err := rpc.Unmarshal(&resp, &failedTxs); err != nil {
					//	log.Error("[MoniterFailedDepositTransfer] Unmarshal getfaileddeposittransactions responce error")
					//	break
					//}

					//Add Test Data
					txid := "de5a9ce6542a7ff603c6cbe38b31f7115b8e3e0a6d76da16630f13c27154ac3d"
					amount := common.Fixed64(1000001000)
					cross := common.Fixed64(1000000000)
					id, _ := common.Uint256FromHexString(txid)
					failedTxs = append(failedTxs, base.FailedDepositTx{
						Txid: id,
						DepositInfo: &base.DepositInfo{
							DepositAssets: []*base.DepositAssets{
								{
									TargetAddress:    "EWY9yB7kreywqjesdaU52eSnbRDBNEDCTy",
									Amount:           &amount,
									CrossChainAmount: &cross,
								},
							},
						},
					})

					//fullfil target address
					//for _ , ftx := range failedTxs {
					//	cachedTx , err := store.DbCache.MainChainStore.GetMainChainTxsFromHashes([]string{ftx.Txid.String()},cfg.GenesisBlockAddress)
					//	if err != nil || len(cachedTx) == 0 {
					//		log.Error("[MoniterFailedDepositTransfer] warning can not find cached tx ")
					//		return
					//	}
					//	referIndex := cachedTx[0].MainChainTransaction.Inputs[0].Previous.Index
					//	referTxid := cachedTx[0].MainChainTransaction.Inputs[0].Previous.TxID
					//	main := &MainChainFuncImpl{}
					//	targetAddress ,err := main.GetReferenceAddress(referTxid.String(), int(referIndex))
					//	if err != nil {
					//		log.Error("[MoniterFailedDepositTransfer] get target address failed " , err.Error())
					//		return
					//	}
					//	for _, as :=range ftx.DepositInfo.DepositAssets {
					//		as.TargetAddress = targetAddress
					//	}
					//}

					if !ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfMain() {
						log.Warn("[MoniterFailedDepositTransfer] i am not onduty")
						break
					}

					err := curr.SendFailedDepositTxs(failedTxs)
					if err != nil {
						log.Error("[MoniterFailedDepositTransfer] CreateAndBroadcastWithdrawProposal failed" , err.Error())
						break
					}
				}
			}
			log.Info("End Monitor Failed Deposit Transfer")
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
