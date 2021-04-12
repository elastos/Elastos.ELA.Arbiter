package sidechain

import (
	"bytes"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/elanet/pact"
)

type SideChainImpl struct {
	mux sync.Mutex

	Key           string
	CurrentConfig *config.SideNodeConfig
	DoneSmallCrs  map[string]bool
}

func (sc *SideChainImpl) IsSendSmallCrxTx(tx string) bool {
	_, ok := sc.DoneSmallCrs[tx]
	return ok
}

func (sc *SideChainImpl) GetKey() string {
	return sc.Key
}

func (sc *SideChainImpl) GetCurrentConfig() *config.SideNodeConfig {
	return sc.getCurrentConfig()
}

func (sc *SideChainImpl) getCurrentConfig() *config.SideNodeConfig {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	if sc.CurrentConfig == nil {
		for _, sideConfig := range config.Parameters.SideNodeList {
			if sc.GetKey() == sideConfig.GenesisBlockAddress {
				sc.CurrentConfig = sideConfig
				break
			}
		}
	}
	return sc.CurrentConfig
}

func (sc *SideChainImpl) GetExchangeRate() (float64, error) {
	con := sc.getCurrentConfig()
	if con == nil {
		return 0, errors.New("get exchange rate failed, side chain has no config")
	}
	if sc.getCurrentConfig().ExchangeRate <= 0 {
		return 0, errors.New("get exchange rate failed, invalid exchange rate")
	}

	return sc.getCurrentConfig().ExchangeRate, nil
}

func (sc *SideChainImpl) GetCurrentHeight() (uint32, error) {
	return rpc.GetCurrentHeight(sc.getCurrentConfig().Rpc)
}

func (sc *SideChainImpl) GetBlockByHeight(height uint32) (*base.BlockInfo, error) {
	return rpc.GetBlockByHeight(height, sc.getCurrentConfig().Rpc)
}

func (sc *SideChainImpl) SendTransaction(txHash *common.Uint256) (rpc.Response, error) {
	log.Info("[Rpc-sendtransactioninfo] Deposit transaction to side chain：", sc.CurrentConfig.Rpc.IpAddress, ":", sc.CurrentConfig.Rpc.HttpJsonPort)
	response, err := rpc.CallAndUnmarshalResponse("sendrechargetransaction", rpc.Param("txid", txHash.String()), sc.CurrentConfig.Rpc)
	if err != nil {
		return rpc.Response{}, err
	}
	log.Info("[Rpc-sendtransactioninfo] Deposit transaction finished")

	if response.Error != nil {
		log.Info("response: ", response.Error.Message)
	} else {
		log.Info("response:", response)
	}

	return response, nil
}

func (sc *SideChainImpl) SendSmallCrossTransaction(tx string, signature []byte, hash string) (rpc.Response, error) {
	log.Info("[Rpc-SendSmallCrossTransaction] Deposit transaction to side chain：", sc.CurrentConfig.Rpc.IpAddress, ":", sc.CurrentConfig.Rpc.HttpJsonPort)
	response, err := rpc.CallAndUnmarshalResponse("sendsmallcrosstransaction", rpc.Param("signature", hex.EncodeToString(signature)).Add("rawTx", tx).Add("txHash", hash), sc.CurrentConfig.Rpc)
	if err != nil {
		return rpc.Response{}, err
	}
	log.Info("[Rpc-SendSmallCrossTransaction] Deposit transaction finished")

	if response.Error != nil {
		log.Info("response: ", response.Error.Message)
	} else if r, ok := response.Result.(bool); ok && r {
		sc.DoneSmallCrs[hash] = true
		log.Info("response:", response)
	}

	return response, nil
}

func (sc *SideChainImpl) GetAccountAddress() string {
	return sc.GetKey()
}

func (sc *SideChainImpl) OnUTXOChanged(withdrawTxs []*base.WithdrawTx, blockHeight uint32) error {
	if len(withdrawTxs) == 0 {
		return errors.New("[OnUTXOChanged] received withdrawTx, but size is 0")
	}

	var txs []*base.SideChainTransaction
	for _, withdrawTx := range withdrawTxs {
		buf := new(bytes.Buffer)
		if err := withdrawTx.Serialize(buf); err != nil {
			log.Error("[OnUTXOChanged] received withdrawTx, but is invalid tx,", err.Error())
			continue
		}

		txs = append(txs, &base.SideChainTransaction{
			TransactionHash:     withdrawTx.Txid.String(),
			GenesisBlockAddress: sc.GetKey(),
			Transaction:         buf.Bytes(),
			BlockHeight:         blockHeight,
		})
	}

	if err := store.DbCache.SideChainStore.AddSideChainTxs(txs); err != nil {
		return err
	}

	log.Info("[OnUTXOChanged] find ", len(txs), "withdraw transaction, add into db cache")
	return nil
}

func (sc *SideChainImpl) OnIllegalEvidenceFound(evidence *payload.SidechainIllegalData) error {
	arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().BroadcastSidechainIllegalData(evidence)
	return nil
}

func (sc *SideChainImpl) StartSideChainMining() {
	if sc.CurrentConfig.PowChain {
		log.Info("[OnDutyChanged] Start side chain mining: genesis address [", sc.Key, "]")
		sideauxpow.StartSideChainMining(sc.CurrentConfig)
	} else {
		log.Debug("[StartSideChainMining] side chain is not pow chain, no need to mining")
	}
}

func (sc *SideChainImpl) SubmitAuxpow(genesishash string, blockhash string, submitauxpow string) error {
	return sideauxpow.SubmitAuxpow(genesishash, blockhash, submitauxpow)
}

func (sc *SideChainImpl) UpdateLastNotifySideMiningHeight(genesisBlockHash common.Uint256) {
	sideauxpow.UpdateLastNotifySideMiningHeight(genesisBlockHash)
}

func (sc *SideChainImpl) UpdateLastSubmitAuxpowHeight(genesisBlockHash common.Uint256) {
	sideauxpow.UpdateLastSubmitAuxpowHeight(genesisBlockHash)
}

func (sc *SideChainImpl) GetExistDepositTransactions(txs []string) ([]string, error) {
	receivedTxs, err := rpc.GetExistDepositTransactions(txs, sc.CurrentConfig.Rpc)
	if err != nil {
		return nil, err
	}
	return receivedTxs, nil
}

func (sc *SideChainImpl) GetWithdrawTransaction(txHash string) (*base.WithdrawTxInfo, error) {
	txInfo, err := rpc.GetTransactionInfoByHash(txHash, sc.CurrentConfig.Rpc)
	if err != nil {
		return nil, err
	}

	return txInfo, nil
}

func (sc *SideChainImpl) GetIllegalDeositTransaction(txHash string) (bool, error) {
	currHeight, err := sc.GetCurrentHeight()
	if err != nil {
		return false, err
	}
	exist, err := rpc.GetDepositTransactionInfoByHash(txHash, sc.CurrentConfig.Rpc, currHeight)
	if err != nil {
		return false, err
	}

	return exist, nil
}

func (sc *SideChainImpl) CheckIllegalEvidence(evidence *base.SidechainIllegalDataInfo) (bool, error) {
	return rpc.CheckIllegalEvidence(evidence, sc.CurrentConfig.Rpc)
}

func (sc *SideChainImpl) CheckIllegalDepositTx(depositTxs []common.Uint256) (bool, error) {
	return rpc.CheckIllegalDepositTx(depositTxs, sc.CurrentConfig.Rpc)
}

func (sc *SideChainImpl) SendFailedDepositTxs(tx []base.FailedDepositTx) error {
	return sc.CreateAndBroadcastFailedDepositTxsProposal(tx)
}

func (sc *SideChainImpl) SendCachedWithdrawTxs() {
	log.Info("[SendCachedWithdrawTxs] start")
	defer log.Info("[SendCachedWithdrawTxs] end")

	txHashes, blockHeights, err := store.DbCache.SideChainStore.GetAllSideChainTxHashesAndHeights(sc.GetKey())
	if err != nil {
		log.Errorf("[SendCachedWithdrawTxs] %s", err.Error())
		return
	}

	if len(txHashes) == 0 {
		log.Info("No cached withdraw transaction need to send")
		return
	}

	if len(txHashes) > config.Parameters.MaxTxsPerWithdrawTx {
		txHashes = txHashes[:config.Parameters.MaxTxsPerWithdrawTx]
	}

	receivedTxs, err := rpc.GetExistWithdrawTransactions(txHashes)
	if err != nil {
		log.Errorf("[SendCachedWithdrawTxs] %s", err.Error())
		return
	}

	unsolvedTxs, _ := base.SubstractTransactionHashesAndBlockHeights(txHashes, blockHeights, receivedTxs)
	if len(unsolvedTxs) != 0 {
		err := sc.CreateAndBroadcastWithdrawProposal(unsolvedTxs)
		if err != nil {
			log.Error("[ReceiveSendLastArbiterUsedUtxos] CreateAndBroadcastWithdrawProposal failed")
		}
	}

	if len(receivedTxs) != 0 {
		err = store.DbCache.SideChainStore.RemoveSideChainTxs(receivedTxs)
		if err != nil {
			log.Errorf("[SendCachedWithdrawTxs] %s", err.Error())
			return
		}

		err = store.FinishedTxsDbCache.AddSucceedWithdrawTxs(receivedTxs)
		if err != nil {
			log.Errorf("[SendCachedWithdrawTxs] %s", err.Error())
			return
		}
	}
}

func (sc *SideChainImpl) CreateAndBroadcastWithdrawProposal(txnHashes []string) error {
	unsolvedTransactions, err := store.DbCache.SideChainStore.GetSideChainTxsFromHashes(txnHashes)
	if err != nil {
		return err
	}

	if len(unsolvedTransactions) == 0 {
		return nil
	}

	targetTransactions := make([]*base.WithdrawTx, 0)
	for _, tx := range unsolvedTransactions {
		if len(tx.WithdrawInfo.WithdrawAssets) != 0 {
			targetTransactions = append(targetTransactions, tx)
		}
	}

	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()

	var wTx *types.Transaction
	var targetIndex int
	for i := 0; i < len(targetTransactions); {
		i += 100
		targetIndex = len(targetTransactions)
		if targetIndex > i {
			targetIndex = i
		}

		tx := currentArbitrator.CreateWithdrawTransaction(
			targetTransactions[:targetIndex], sc, &arbitrator.MainChainFuncImpl{})
		if tx == nil {
			continue
		}
		if tx.GetSize() < int(pact.MaxBlockContextSize) {
			wTx = tx
		}
	}

	if wTx == nil {
		return errors.New("[CreateAndBroadcastWithdrawProposal] failed")
	}
	currentArbitrator.BroadcastWithdrawProposal(wTx)
	log.Info("[CreateAndBroadcastWithdrawProposal] transactions count: ", targetIndex)

	return nil
}

func (sc *SideChainImpl) CreateAndBroadcastFailedDepositTxsProposal(failedTxs []base.FailedDepositTx) error {

	if len(failedTxs) == 0 {
		return nil
	}
	log.Info("1111")
	targetTransactions := make([]*base.FailedDepositTx, 0)
	for _, tx := range failedTxs {
		if len(tx.DepositInfo.DepositAssets) != 0 {
			targetTransactions = append(targetTransactions, &tx)
		}
	}

	log.Info("Tx targetaddress ", failedTxs[0].DepositInfo.DepositAssets[0].TargetAddress)
	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()
	log.Info("11111")
	var wTx *types.Transaction
	var targetIndex int
	for i := 0; i < len(targetTransactions); {
		i += 100
		targetIndex = len(targetTransactions)
		if targetIndex > i {
			targetIndex = i
		}

		tx := currentArbitrator.CreateFailedDepositTransaction(
			targetTransactions[:targetIndex], sc, &arbitrator.MainChainFuncImpl{})
		log.Info("sdfasdasdfaf")
		if tx == nil {
			continue
		}
		log.Info("sdfasdasdfaf111")
		log.Info("Before serialze ")
		log.Info("Before serialze ", tx.String())
		//if tx.GetSize() < int(pact.MaxBlockContextSize) {
		wTx = tx
		//}

		//log.Info("Before serialze " , wTx.String())

		log.Info("Before serialze ", "111 ")
		buf := new(bytes.Buffer)
		if err := wTx.Serialize(buf); err != nil {
			log.Warn("tx serialize error ", err.Error())
		}

		log.Info(hex.EncodeToString(buf.Bytes()))
		log.Info("222221111")
		var txD types.Transaction
		err := txD.Deserialize(bytes.NewReader(buf.Bytes()))
		if err != nil {
			log.Warn("tx deserialize error ", err.Error(), txD.String())
		}
		log.Info("2222211113333")
		testPayload, k := txD.Payload.(*payload.ReturnSideChainDepositCoin)
		if !k {
			log.Error("payload deserialize error")
		} else {
			log.Info(testPayload.Height, testPayload.GenesisBlockAddress, testPayload.DepositTxs[0].String())
		}
		log.Info("22222111133332222")
	}
	log.Info("111111")
	if wTx == nil {
		return errors.New("[CreateAndBroadcastWithdrawProposal] failed")
	}
	log.Info("22222111133331111")
	currentArbitrator.BroadcastWithdrawProposal(wTx)
	log.Info("[CreateAndBroadcastWithdrawProposal] transactions count: ", targetIndex)

	return nil
}
