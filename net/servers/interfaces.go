package servers

import (
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/complain"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	. "github.com/elastos/Elastos.ELA.Arbiter/errors"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
	. "github.com/elastos/Elastos.ELA.Arbiter/store"

	. "github.com/elastos/Elastos.ELA.Utility/common"
)

func SubmitComplain(param Params) map[string]interface{} {
	if !checkParam(param, "fromaddress", "transactionhash") {
		return ResponsePack(InvalidParams, "")
	}

	transactionHash := param["transactionhash"].(string)
	txHashBytes, _ := HexStringToBytes(transactionHash)
	txHashBytes = BytesReverse(txHashBytes)
	txHash, err := Uint256FromBytes(txHashBytes)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	fromAddress := param["fromaddress"].(string)
	blockHashItem, ok := param["chaingenesisblockhash"]
	if !ok {
		blockHashItem = ""
	}

	content, err := complain.ComplainSolver.AcceptComplain(fromAddress, blockHashItem.(string), *txHash)
	if err != nil {
		return ResponsePack(InvalidTransaction, "")
	}

	if err = complain.ComplainSolver.BroadcastComplainSolving(content); err != nil {
		return ResponsePack(InternalError, "")
	}

	return ResponsePack(Success, "")
}

func GetComplainStatus(param Params) map[string]interface{} {
	if !checkParam(param, "transactionhash") {
		return ResponsePack(InvalidParams, "")
	}

	transactionHash := param["transactionhash"].(string)
	txHashBytes, _ := HexStringToBytes(transactionHash)
	txHashBytes = BytesReverse(txHashBytes)
	txHash, err := Uint256FromBytes(txHashBytes)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	return ResponsePack(Success, complain.ComplainSolver.GetComplainStatus(*txHash))
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
		Version                      int           `json:"version"`
		SideChainMonitorScanInterval time.Duration `json:"SideChainMonitorScanInterval"`
		ClearTransactionInterval     time.Duration `json:"ClearTransactionInterval"`
		MinReceivedUsedUtxoMsgNumber uint32        `json:"MinReceivedUsedUtxoMsgNumber"`
		MinOutbound                  int           `json:"MinOutbound"`
		MaxConnections               int           `json:"MaxConnections"`
		SideAuxPowFee                int           `json:"SideAuxPowFee"`
		MinThreshold                 int           `json:"MinThreshold"`
		DepositAmount                int           `json:"DepositAmount"`
	}{
		Version: config.Parameters.Version,
		SideChainMonitorScanInterval: config.Parameters.SideChainMonitorScanInterval,
		ClearTransactionInterval:     config.Parameters.ClearTransactionInterval,
		MinReceivedUsedUtxoMsgNumber: config.Parameters.MinReceivedUsedUtxoMsgNumber,
		MinOutbound:                  config.Parameters.MinOutbound,
		MaxConnections:               config.Parameters.MaxConnections,
		SideAuxPowFee:                config.Parameters.SideAuxPowFee,
		MinThreshold:                 config.Parameters.MinThreshold,
		DepositAmount:                config.Parameters.DepositAmount,
	}
	return ResponsePack(Success, &Info)
}

func GetSideMiningInfo(param Params) map[string]interface{} {
	genesisBlockHashStr, ok := param.String("hash")
	if !ok {
		return ResponsePack(InvalidParams, "need a string parameter named hash")
	}
	genesisBlockHashBytes, err := HexStringToBytes(genesisBlockHashStr)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid genesis block hash")
	}
	reversedGenesisBlockHashBytes := BytesReverse(genesisBlockHashBytes)
	reversedGenesisBlockHashStr := BytesToHexString(reversedGenesisBlockHashBytes)
	genesisBlockHash, err := Uint256FromHexString(reversedGenesisBlockHashStr)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid genesis block hash")
	}
	lastSendSideMiningHeight, ok := sideauxpow.GetLastSendSideMiningHeight(genesisBlockHash)
	if !ok {
		return ResponsePack(InvalidParams, "genesis block hash not matched")
	}
	lastNotifySideMiningHeight, ok := sideauxpow.GetLastNotifySideMiningHeight(genesisBlockHash)
	if !ok {
		return ResponsePack(InvalidParams, "genesis block hash not matched")
	}
	lastSubmitAuxpowHeight, ok := sideauxpow.GetLastSubmitAuxpowHeight(genesisBlockHash)
	if !ok {
		return ResponsePack(InvalidParams, "genesis block hash not matched")
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
	return ResponsePack(Success, &Info)
}

func GetMainChainBlockHeight(param Params) map[string]interface{} {
	return ResponsePack(Success, DbCache.UTXOStore.CurrentHeight(0))
}

func GetSideChainBlockHeight(param Params) map[string]interface{} {
	genesisBlockHashStr, ok := param.String("hash")
	if !ok {
		return ResponsePack(InvalidParams, "need a string parameter named hash")
	}
	genesisBlockHashBytes, err := HexStringToBytes(genesisBlockHashStr)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid genesis block hash")
	}
	reversedGenesisBlockHashBytes := BytesReverse(genesisBlockHashBytes)
	reversedGenesisBlockHashStr := BytesToHexString(reversedGenesisBlockHashBytes)
	genesisBlockHash, err := Uint256FromHexString(reversedGenesisBlockHashStr)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid genesis block hash")
	}
	address, err := base.GetGenesisAddress(*genesisBlockHash)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid genesis block hash")
	}

	return ResponsePack(Success, DbCache.SideChainStore.CurrentSideHeight(address, 0))
}

func GetFinishedDepositTxs(param Params) map[string]interface{} {
	succeed, ok := param.Bool("succeed")
	if !ok {
		return ResponsePack(InvalidParams, "need a bool parameter named succeed")
	}
	txHashes, genesisAddresses, err := FinishedTxsDbCache.GetDepositTxs(succeed)
	if err != nil {
		return ResponsePack(InvalidParams, "get deposit transactions from finished dbcache failed")
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

	return ResponsePack(Success, &depositTxs)
}

func GetFinishedWithdrawTxs(param Params) map[string]interface{} {
	succeed, ok := param.Bool("succeed")
	if !ok {
		return ResponsePack(InvalidParams, "need a bool parameter named succeed")
	}
	txHashes, err := FinishedTxsDbCache.GetWithdrawTxs(succeed)
	if err != nil {
		return ResponsePack(InvalidParams, "get withdraw transactions from finished dbcache failed")
	}
	withdrawTxs := struct{ Transactions []string }{}

	for _, hash := range txHashes {
		withdrawTxs.Transactions = append(withdrawTxs.Transactions, hash)
	}

	return ResponsePack(Success, &withdrawTxs)
}

func GetGitVersion(param Params) map[string]interface{} {
	return ResponsePack(Success, config.Version)
}

func GetSPVHeight(param Params) map[string]interface{} {
	bestHeader, err := arbitrator.SpvService.HeaderStore().GetBest()
	if err != nil {
		return ResponsePack(InternalError, "get spv best header failed")
	}
	return ResponsePack(Success, bestHeader.Height)
}
