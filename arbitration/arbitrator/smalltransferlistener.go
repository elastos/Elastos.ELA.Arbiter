package arbitrator

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	elatx "github.com/elastos/Elastos.ELA/core/transaction"
)

func MonitorSmallCrossTransfer() {
	for {
		select {
		case <-time.After(time.Second * 1):
			resp, err := rpc.CallAndUnmarshal("getsmallcrosstransfertxs", nil,
				config.Parameters.MainNode.Rpc)
			if err != nil {
				log.Errorf("[Small-Transfer] Unable to call GetSmallCrossTransferTxs rpc ")
				break
			}

			type SmallCrossTransferTx struct {
				RawTx []string `json:"txs"`
			}

			s := SmallCrossTransferTx{}
			if err := rpc.Unmarshal(&resp, &s); err != nil {
				log.Error("[Small-Transfer] Unmarshal GetSmallCrossTransferTxs responce error")
				break
			}

			currentArbitrator, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().(*ArbitratorImpl)
			if !ok {
				log.Error("[Small-Transfer] Unable to get current arbiter")
				break
			}
			var txs []*base.MainChainTransaction
			for _, r := range s.RawTx {
				buf, err := hex.DecodeString(r)
				if err != nil {
					log.Error("[Small-Transfer] Invalid data from GetSmallCrossTransferTxs")
					break
				}
				r := bytes.NewReader(buf)
				txn, err := elatx.GetTransactionByBytes(r)
				if err != nil {
						log.Error("[Small-Transfer] Invalid data from GetSmallCrossTransferTxs")
						break
				}
				err = txn.Deserialize(r)
				if err != nil {
					log.Error("[Small-Transfer] Decode transaction error", err.Error())
					break
				}
				xAddr, err := txn.Outputs()[0].ProgramHash.ToAddress()
				if err != nil {
					log.Error("[Small-Transfer] get xaddress failed ", err.Error())
					break
				}
				side, ok := currentArbitrator.GetChain(xAddr)
				if !ok {
					log.Error("[Small-Transfer] unrecognized xAddr", xAddr)
					break
				}
				if txn.IsSmallTransfer(config.Parameters.SmallCrossTransferThreshold) && side.GetCurrentConfig().SupportQuickRecharge {
					txs = append(txs, &base.MainChainTransaction{
						TransactionHash:     txn.Hash().String(),
						GenesisBlockAddress: xAddr,
						Transaction:         txn,
						Proof:               nil,
					})
				}
			}

			sendingTxs := make(map[string][]*base.SmallCrossTransaction, 0)
			for i := 0; i < len(txs); i++ {
				knownTxs, ok := sendingTxs[txs[i].GenesisBlockAddress]
				if !ok {
					knownTxs = make([]*base.SmallCrossTransaction, 0)
					sendingTxs[txs[i].GenesisBlockAddress] = knownTxs
				}
				buf := new(bytes.Buffer)
				txs[i].Transaction.Serialize(buf)
				signature, err := currentArbitrator.Sign(buf.Bytes())
				if err != nil {
					log.Error("[Small-Transfer] currentArbiter sign error ", err.Error())
					break
				}
				knownTxs = append(knownTxs, &base.SmallCrossTransaction{MainTx: txs[i].Transaction, Signature: signature})
				sendingTxs[txs[i].GenesisBlockAddress] = knownTxs
			}

			for xAddr, knownTxs := range sendingTxs {
				log.Info("[Small-Transfer] find small deposit transaction, create and send deposit transaction, size of txs:", len(knownTxs))
				for index, knownTx := range knownTxs {
					log.Info("[Small-Transfer] tx hash[", index, "]:", knownTx.MainTx.Hash().String())
				}
				ArbitratorGroupSingleton.GetCurrentArbitrator().SendSmallCrossDepositTransactions(knownTxs, xAddr)
			}
		}
	}

}
