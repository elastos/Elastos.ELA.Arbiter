package rpc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common/config"
	"errors"
)

type Response struct {
	Code   int         `json:"code""`
	Result interface{} `json:"result""`
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
	unmarshal(&resp, block)

	return block, nil
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
		fmt.Printf("POST requset: %v\n", err)
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

func unmarshal(result interface{}, target interface{}) error {
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
