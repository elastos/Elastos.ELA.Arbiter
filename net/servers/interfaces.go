package servers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/complain"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/errors"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/common"
)

func SetRegisterSideChainRPCInfo(param Params) map[string]interface{} {
	str, ok := param.String("data")
	if !ok {
		return ResponsePack(errors.InvalidParams, "need a string parameter named data")
	}

	bys, err := common.HexStringToBytes(str)
	if err != nil {
		return ResponsePack(errors.InvalidParams, "hex string to bytes error")
	}
	rpcDetails := &base.RegisterSidechainRpcInfo{}
	err = json.Unmarshal(bys, rpcDetails)
	if err != nil {
		return ResponsePack(errors.InvalidParams, "can not unmarshal bytes")
	}
	arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().OnReceivedRegisteredSideChain(*rpcDetails)
	return ResponsePack(errors.Success, fmt.Sprint(""))
}

func SubmitComplain(param Params) map[string]interface{} {
	if !checkParam(param, "fromaddress", "transactionhash") {
		return ResponsePack(errors.InvalidParams, "")
	}

	transactionHash := param["transactionhash"].(string)
	txHashBytes, _ := common.HexStringToBytes(transactionHash)
	txHashBytes = common.BytesReverse(txHashBytes)
	txHash, err := common.Uint256FromBytes(txHashBytes)
	if err != nil {
		return ResponsePack(errors.InvalidParams, "")
	}

	fromAddress := param["fromaddress"].(string)
	blockHashItem, ok := param["chaingenesisblockhash"]
	if !ok {
		blockHashItem = ""
	}

	content, err := complain.ComplainSolver.AcceptComplain(fromAddress, blockHashItem.(string), *txHash)
	if err != nil {
		return ResponsePack(errors.InvalidTransaction, "")
	}

	if err = complain.ComplainSolver.BroadcastComplainSolving(content); err != nil {
		return ResponsePack(errors.InternalError, "")
	}

	return ResponsePack(errors.Success, "")
}

func GetComplainStatus(param Params) map[string]interface{} {
	if !checkParam(param, "transactionhash") {
		return ResponsePack(errors.InvalidParams, "")
	}

	transactionHash := param["transactionhash"].(string)
	txHashBytes, _ := common.HexStringToBytes(transactionHash)
	txHashBytes = common.BytesReverse(txHashBytes)
	txHash, err := common.Uint256FromBytes(txHashBytes)
	if err != nil {
		return ResponsePack(errors.InvalidParams, "")
	}

	return ResponsePack(errors.Success, complain.ComplainSolver.GetComplainStatus(*txHash))
}

func checkParam(param map[string]interface{}, keys ...string) bool {
	if param == nil {
		return false
	}
	if len(param) < len(keys) {
		return false
	}
	for _, key := range keys {
		value, ok := param[key]
		if !ok {
			return false
		}
		switch value.(type) {
		case string:
		default:
			return false
		}
	}
	return true
}

func GetInfo(param Params) map[string]interface{} {
	Info := struct {
		Version                      uint32        `json:"version"`
		SideChainMonitorScanInterval time.Duration `json:"SideChainMonitorScanInterval"`
		ClearTransactionInterval     time.Duration `json:"ClearTransactionInterval"`
		MinOutbound                  int           `json:"MinOutbound"`
		MaxConnections               int           `json:"MaxConnections"`
		SideAuxPowFee                int           `json:"SideAuxPowFee"`
		MinThreshold                 int           `json:"MinThreshold"`
		DepositAmount                int           `json:"DepositAmount"`
	}{
		Version:                      config.Parameters.Version,
		SideChainMonitorScanInterval: config.Parameters.SideChainMonitorScanInterval,
		ClearTransactionInterval:     config.Parameters.ClearTransactionInterval,
		MinOutbound:                  config.Parameters.MinOutbound,
		MaxConnections:               config.Parameters.MaxConnections,
		SideAuxPowFee:                config.Parameters.SideAuxPowFee,
		MinThreshold:                 config.Parameters.MinThreshold,
		DepositAmount:                config.Parameters.DepositAmount,
	}
	return ResponsePack(errors.Success, &Info)
}

func GetSideMiningInfo(param Params) map[string]interface{} {
	genesisBlockHashStr, ok := param.String("hash")
	if !ok {
		return ResponsePack(errors.InvalidParams, "need a string parameter named hash")
	}
	genesisBlockHashBytes, err := common.HexStringToBytes(genesisBlockHashStr)
	if err != nil {
		return ResponsePack(errors.InvalidParams, "invalid genesis block hash")
	}
	reversedGenesisBlockHashBytes := common.BytesReverse(genesisBlockHashBytes)
	reversedGenesisBlockHashStr := common.BytesToHexString(reversedGenesisBlockHashBytes)
	genesisBlockHash, err := common.Uint256FromHexString(reversedGenesisBlockHashStr)
	if err != nil {
		return ResponsePack(errors.InvalidParams, "invalid genesis block hash")
	}
	lastSendSideMiningHeight, ok := sideauxpow.GetLastSendSideMiningHeight(genesisBlockHash)
	if !ok {
		return ResponsePack(errors.InvalidParams, "genesis block hash not matched")
	}
	lastNotifySideMiningHeight, ok := sideauxpow.GetLastNotifySideMiningHeight(genesisBlockHash)
	if !ok {
		return ResponsePack(errors.InvalidParams, "genesis block hash not matched")
	}
	lastSubmitAuxpowHeight, ok := sideauxpow.GetLastSubmitAuxpowHeight(genesisBlockHash)
	if !ok {
		return ResponsePack(errors.InvalidParams, "genesis block hash not matched")
	}
	Info := struct {
		LastSendSideMiningHeight   uint32
		LastNotifySideMiningHeight uint32
		LastSubmitAuxpowHeight     uint32
	}{
		LastSendSideMiningHeight:   lastSendSideMiningHeight,
		LastNotifySideMiningHeight: lastNotifySideMiningHeight,
		LastSubmitAuxpowHeight:     lastSubmitAuxpowHeight,
	}
	return ResponsePack(errors.Success, &Info)
}

func GetMainChainBlockHeight(param Params) map[string]interface{} {
	return ResponsePack(errors.Success, store.DbCache.MainChainStore.CurrentHeight(0))
}

func GetSideChainBlockHeight(param Params) map[string]interface{} {
	genesisBlockHashStr, ok := param.String("hash")
	if !ok {
		return ResponsePack(errors.InvalidParams, "need a string parameter named hash")
	}
	genesisBlockHashBytes, err := common.HexStringToBytes(genesisBlockHashStr)
	if err != nil {
		return ResponsePack(errors.InvalidParams, "invalid genesis block hash")
	}
	reversedGenesisBlockHashBytes := common.BytesReverse(genesisBlockHashBytes)
	reversedGenesisBlockHashStr := common.BytesToHexString(reversedGenesisBlockHashBytes)
	genesisBlockHash, err := common.Uint256FromHexString(reversedGenesisBlockHashStr)
	if err != nil {
		return ResponsePack(errors.InvalidParams, "invalid genesis block hash")
	}
	address, err := base.GetGenesisAddress(*genesisBlockHash)
	if err != nil {
		return ResponsePack(errors.InvalidParams, "invalid genesis block hash")
	}

	return ResponsePack(errors.Success, store.DbCache.SideChainStore.CurrentSideHeight(address, 0))
}

func GetFinishedDepositTxs(param Params) map[string]interface{} {
	succeed, ok := param.Bool("succeed")
	if !ok {
		return ResponsePack(errors.InvalidParams, "need a bool parameter named succeed")
	}
	txHashes, genesisAddresses, err := store.FinishedTxsDbCache.GetDepositTxs(succeed)
	if err != nil {
		return ResponsePack(errors.InvalidParams, "get deposit transactions from finished dbcache failed")
	}
	type depositTx struct {
		Hash                string
		GenesisBlockAddress string
	}
	depositTxs := struct {
		Transactions []depositTx
	}{}

	for i := 0; i < len(txHashes); i++ {
		depositTxs.Transactions = append(depositTxs.Transactions,
			depositTx{
				Hash:                txHashes[i],
				GenesisBlockAddress: genesisAddresses[i],
			})
	}

	return ResponsePack(errors.Success, &depositTxs)
}

func GetFinishedWithdrawTxs(param Params) map[string]interface{} {
	succeed, ok := param.Bool("succeed")
	if !ok {
		return ResponsePack(errors.InvalidParams, "need a bool parameter named succeed")
	}
	txHashes, err := store.FinishedTxsDbCache.GetWithdrawTxs(succeed)
	if err != nil {
		return ResponsePack(errors.InvalidParams, "get withdraw transactions from finished dbcache failed")
	}
	withdrawTxs := struct{ Transactions []string }{}

	for _, hash := range txHashes {
		withdrawTxs.Transactions = append(withdrawTxs.Transactions, hash)
	}

	return ResponsePack(errors.Success, &withdrawTxs)
}

func GetGitVersion(param Params) map[string]interface{} {
	return ResponsePack(errors.Success, config.Version)
}

func GetSPVHeight(param Params) map[string]interface{} {
	bestHeader, err := arbitrator.SpvService.HeaderStore().GetBest()
	if err != nil {
		return ResponsePack(errors.InternalError, "get spv best header failed")
	}
	return ResponsePack(errors.Success, bestHeader.Height)
}

func GetArbiterPeersInfo(params Params) map[string]interface{} {
	type peerInfo struct {
		PublicKey string `json:"publickey"`
		IP        string `json:"ip"`
		ConnState string `json:"connstate"`
	}
	peers := cs.P2PClientSingleton.DumpArbiterPeersInfo()
	result := make([]peerInfo, 0)
	for _, p := range peers {
		result = append(result, peerInfo{
			PublicKey: hex.EncodeToString(p.PID[:]),
			IP:        p.Addr,
			ConnState: p.State.String(),
		})
	}
	return ResponsePack(errors.Success, result)
}
