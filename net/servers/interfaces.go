package servers

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/complain"
	. "github.com/elastos/Elastos.ELA.Arbiter/errors"
	. "github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA.SideChain/common"
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

func GetMainChainBlockHeight(param Params) map[string]interface{} {
	return ResponsePack(Success, DbCache.UTXOStore.CurrentHeight(0))
}

func GetSideChainBlockHeightByGenesisAddress(param Params) map[string]interface{} {
	addr, ok := param.String("addr")
	if !ok {
		return ResponsePack(InvalidParams, "need a string parameter named addr")
	}
	return ResponsePack(Success, DbCache.SideChainStore.CurrentSideHeight(addr, 0))
}

func GetSideChainBlockHeightByGenesisBlockHash(param Params) map[string]interface{} {
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
	address, err := common.GetGenesisAddress(*genesisBlockHash)
	if err != nil {
		return ResponsePack(InvalidParams, "invalid genesis block hash")
	}

	return ResponsePack(Success, DbCache.SideChainStore.CurrentSideHeight(address, 0))
}
