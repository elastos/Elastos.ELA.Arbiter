package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	elaCore "github.com/elastos/Elastos.ELA/core"
)

type Response struct {
	Code   int         `json:"code""`
	Result interface{} `json:"result""`
}

type ArbitratorGroupInfo struct {
	OnDutyArbitratorIndex int
	Arbitrators           []string
}

type BlockTransactions struct {
	Hash         string
	Height       uint32
	Transactions []*TransactionInfo
}

func GetArbitratorGroupInfoByHeight(height uint32) (*ArbitratorGroupInfo, error) {
	resp, err := CallAndUnmarshal("getarbitratorgroupbyheight", Param("height", height), config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}
	groupInfo := &ArbitratorGroupInfo{}
	Unmarshal(&resp, groupInfo)

	return groupInfo, nil
}

func GetCurrentHeight(config *config.RpcConfig) (uint32, error) {
	result, err := CallAndUnmarshal("getcurrentheight", nil, config)
	if err != nil {
		return 0, err
	}
	return uint32(result.(float64)), nil
}

func GetBlockByHeight(height uint32, config *config.RpcConfig) (*BlockInfo, error) {
	resp, err := CallAndUnmarshal("getblockbyheight", Param("height", height), config)
	if err != nil {
		return nil, err
	}
	block := &BlockInfo{}
	Unmarshal(&resp, block)

	return block, nil
}

func GetDestroyedTransactionByHeight(height uint32, config *config.RpcConfig) (*BlockTransactions, error) {
	resp, err := CallAndUnmarshal("getdestroyedtransactions", Param("height", height), config)
	if err != nil {
		return nil, err
	}
	transactions, err := GetBlockTransactions(resp)
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

func IsTransactionExist(transactionHash string, config *config.RpcConfig) (bool, error) {
	_, err := CallAndUnmarshal("getrawtransaction", Param("hash", transactionHash), config)
	if err != nil {
		return false, err
	}

	return true, nil
}

func Call(method string, params map[string]string, config *config.RpcConfig) ([]byte, error) {
	url := "http://" + config.IpAddress + ":" + strconv.Itoa(config.HttpJsonPort)
	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}

	//log.Trace("RPC call:", string(data))
	resp, err := http.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		log.Info("POST requset: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	//log.Trace("RPC resp:", string(body))

	return body, nil
}

func GetBlockTransactions(resp interface{}) (*BlockTransactions, error) {
	transactions := &BlockTransactions{}
	Unmarshal(&resp, transactions)

	for _, txInfo := range transactions.Transactions {
		switch txInfo.TxType {
		case elaCore.RegisterAsset:
			assetInfo := &RegisterAssetInfo{}
			err := Unmarshal(&txInfo.Payload, assetInfo)
			if err == nil {
				txInfo.Payload = assetInfo
			}
		case elaCore.TransferAsset:
			assetInfo := &TransferAssetInfo{}
			err := Unmarshal(&txInfo.Payload, assetInfo)
			if err == nil {
				txInfo.Payload = assetInfo
			}
		case elaCore.IssueToken:
			assetInfo := &IssueTokenInfo{}
			err := Unmarshal(&txInfo.Payload, assetInfo)
			if err == nil {
				txInfo.Payload = assetInfo
			}
		case elaCore.TransferCrossChainAsset:
			assetInfo := &TransferCrossChainAssetInfo{}
			err := Unmarshal(&txInfo.Payload, assetInfo)
			if err == nil {
				txInfo.Payload = assetInfo
			}
		default:
			return nil, errors.New("GetBlockTransactions: Unknown payload type")
		}
	}
	return transactions, nil
}

func CallAndUnmarshal(method string, params map[string]string, config *config.RpcConfig) (interface{}, error) {
	body, err := Call(method, params, config)
	if err != nil {
		return nil, err
	}

	resp := Response{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return string(body), nil
	}

	if resp.Code != 0 {
		return nil, errors.New(fmt.Sprint(resp.Result))
	}

	return resp.Result, nil
}

func Unmarshal(result interface{}, target interface{}) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, target)
	if err != nil {
		return err
	}
	return nil
}
