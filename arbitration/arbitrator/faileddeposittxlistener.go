package arbitrator

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"time"
)

func MoniterFailedDepositTransfer() {
	for {
		select {
		case <-time.After(time.Second * 30):
			log.Info("Start Monitor Failed Deposit Transfer")
			log.Info("Start Monitor Failed Deposit Transfer 111")
			currentArbitrator, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().(*ArbitratorImpl)
			log.Info("Start Monitor Failed Deposit Transfer 222")
			if !ok {
				log.Error("[MoniterFailedDepositTransfer] Unable to get current arbiter")
				break
			}
			log.Info("1")
			for _, curr := range currentArbitrator.GetAllChains() {
				cfg := curr.GetCurrentConfig()
				if cfg.SupportQuickRecharge {
					log.Info("11")
					param := make(map[string]interface{})
					height, err := curr.GetCurrentHeight()
					if err != nil {
						log.Errorf("[MoniterFailedDepositTransfer] Unable to call get current height")
						break
					}
					param["height"] = height
					resp, err := rpc.CallAndUnmarshal("getfaileddeposittransactions", param,
						cfg.Rpc)
					if err != nil {
						log.Errorf("[MoniterFailedDepositTransfer] Unable to call getfaileddeposittransactions rpc ")
						break
					}
					var fTxs []string
					if err := rpc.Unmarshal(&resp, &fTxs); err != nil {
						log.Error("[MoniterFailedDepositTransfer] Unmarshal getfaileddeposittransactions responce error")
						break
					}
					var failedTxs []base.FailedDepositTx
					for _, tx := range fTxs {
						originTx, err := rpc.GetTransaction(tx, config.Parameters.MainNode.Rpc)
						if err != nil {
							log.Errorf(err.Error())
							break
						}
						referTxid := originTx.Inputs[0].Previous.TxID
						referIndex := originTx.Inputs[0].Previous.Index

						referTxn, err := rpc.GetTransaction(referTxid.String(), config.Parameters.MainNode.Rpc)
						if err != nil {
							log.Errorf(err.Error())
							break
						}
						originHash := originTx.Hash()
						payload, ok := originTx.Payload.(*payload.TransferCrossChainAsset)
						if !ok {
							log.Errorf("Invalid payload type need TransferCrossChainAsset")
							break
						}
						address := referTxn.Outputs[referIndex].ProgramHash.String()
						for i, cca := range payload.CrossChainAmounts {
							idx := payload.OutputIndexes[i]
							amount := originTx.Outputs[idx].Value
							failedTxs = append(failedTxs, base.FailedDepositTx{
								Txid: &originHash,
								DepositInfo: &base.DepositInfo{
									DepositAssets: []*base.DepositAssets{
										{
											TargetAddress:    address,
											Amount:           &amount,
											CrossChainAmount: &cca,
										},
									},
								}})
						}
					}

					// need to form the testdata struct.failedTxs according to fTxs from sidechain
					//var failedTxs []base.FailedDepositTx

					//Add Test Data
					//txid := "de5a9ce6542a7ff603c6cbe38b31f7115b8e3e0a6d76da16630f13c27154ac3d"
					//amount := common.Fixed64(1000001000)
					//cross := common.Fixed64(1000000000)
					//id, _ := common.Uint256FromHexString(txid)
					//failedTxs = append(failedTxs, base.FailedDepositTx{
					//	Txid: id,
					//	DepositInfo: &base.DepositInfo{
					//		DepositAssets: []*base.DepositAssets{
					//			{
					//				TargetAddress:    "EWY9yB7kreywqjesdaU52eSnbRDBNEDCTy",
					//				Amount:           &amount,
					//				CrossChainAmount: &cross,
					//			},
					//		},
					//	},
					//})

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
					log.Info("111")
					err = curr.SendFailedDepositTxs(failedTxs)
					if err != nil {
						log.Error("[MoniterFailedDepositTransfer] CreateAndBroadcastWithdrawProposal failed", err.Error())
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
