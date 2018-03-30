package servers

import (
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/complain"
	. "github.com/elastos/Elastos.ELA.Arbiter/common"
	. "github.com/elastos/Elastos.ELA.Arbiter/errors"
)

func SubmitComplain(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "fromaddress", "transactionhash") {
		return ResponsePack(InvalidParams, "")
	}

	transactionHash := param["transactionhash"].(string)
	txHashBytes, _ := HexStringToBytesReverse(transactionHash)
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

func GetComplainStatus(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "transactionhash") {
		return ResponsePack(InvalidParams, "")
	}

	transactionHash := param["transactionhash"].(string)
	txHashBytes, _ := HexStringToBytesReverse(transactionHash)
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
