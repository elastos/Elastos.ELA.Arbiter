package rpc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
)

type Response struct {
	ID      int64       `json:"id"`
	Version string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	*Error  `json:"error"`
}

type Error struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

type ArbitratorGroupInfo struct {
	OnDutyArbitratorIndex int
	Arbitrators           []string
}

func GetActiveDposPeers(height uint32) (result []peer.PID, err error) {
	if height+1 < config.Parameters.CRCOnlyDPOSHeight {
		for _, a := range config.Parameters.OriginCrossChainArbiters {
			var id peer.PID
			pk, err := common.HexStringToBytes(a)
			if err != nil {
				return nil, err
			}

			copy(id[:], pk)
			result = append(result, id)
		}

		return result, nil
	}

	if height+1 >= config.Parameters.CRCOnlyDPOSHeight &&
		height < config.Parameters.CRClaimDPOSNodeStartHeight {
		for _, a := range config.Parameters.CRCCrossChainArbiters {
			var id peer.PID
			pk, err := common.HexStringToBytes(a)
			if err != nil {
				return nil, err
			}

			copy(id[:], pk)
			result = append(result, id)
		}

		return result, nil
	}

	var rpcMethod string
	if height < config.Parameters.DPOSNodeCrossChainHeight {
		rpcMethod = "getcrcpeersinfo"
	} else {
		rpcMethod = "getcrosschainpeersinfo"
	}
	resp, err := CallAndUnmarshal(rpcMethod, nil,
		config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}

	type peerInfo struct {
		NodePublicKeys []string `json:"nodepublickeys"`
	}

	peers := peerInfo{}
	if err := Unmarshal(&resp, &peers); err != nil {
		return nil, err
	}

	for _, v := range peers.NodePublicKeys {
		var id peer.PID
		pk, err := common.HexStringToBytes(v)
		if err != nil {
			return nil, err
		}

		copy(id[:], pk)
		result = append(result, id)
	}

	return result, nil
}

func GetArbitratorGroupInfoByHeight(height uint32) (*ArbitratorGroupInfo, error) {
	groupInfo := &ArbitratorGroupInfo{
		Arbitrators: make([]string, 0),
	}
	if height+1 < config.Parameters.CRCOnlyDPOSHeight {
		for _, a := range config.Parameters.OriginCrossChainArbiters {
			groupInfo.Arbitrators = append(groupInfo.Arbitrators, a)
		}
		groupInfo.OnDutyArbitratorIndex = int(height) % len(groupInfo.Arbitrators)
		return groupInfo, nil
	}

	if height+1 >= config.Parameters.CRCOnlyDPOSHeight &&
		height < config.Parameters.CRClaimDPOSNodeStartHeight {
		for _, a := range config.Parameters.CRCCrossChainArbiters {
			groupInfo.Arbitrators = append(groupInfo.Arbitrators, a)
		}
		sort.Strings(groupInfo.Arbitrators)
		groupInfo.OnDutyArbitratorIndex = int(height-config.Parameters.CRCOnlyDPOSHeight+1) % len(groupInfo.Arbitrators)
		return groupInfo, nil
	}

	resp, err := CallAndUnmarshal("getarbitratorgroupbyheight",
		Param("height", height), config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}
	if err := Unmarshal(&resp, groupInfo); err != nil {
		return nil, err
	}

	return groupInfo, nil
}

func GetCurrentHeight(config *config.RpcConfig) (uint32, error) {
	result, err := CallAndUnmarshal("getblockcount", nil, config)
	if err != nil {
		return 0, err
	}
	if count, ok := result.(float64); ok && count >= 1 {
		return uint32(count) - 1, nil
	}
	return 0, errors.New("[GetCurrentHeight] invalid count")
}

func GetBlockByHeight(height uint32, config *config.RpcConfig) (*base.BlockInfo, error) {
	resp, err := CallAndUnmarshal("getblockbyheight", Param("height", height), config)
	if err != nil {
		return nil, err
	}
	block := &base.BlockInfo{}
	Unmarshal(&resp, block)

	return block, nil
}

func GetBlockByHash(hash *common.Uint256, config *config.RpcConfig) (*base.BlockInfo, error) {
	hashBytes, err := common.HexStringToBytes(hash.String())
	if err != nil {
		return nil, err
	}
	reversedHashBytes := common.BytesReverse(hashBytes)
	reversedHashStr := common.BytesToHexString(reversedHashBytes)

	resp, err := CallAndUnmarshal("getblock",
		Param("blockhash", reversedHashStr).Add("verbosity", 2), config)
	if err != nil {
		return nil, err
	}
	block := &base.BlockInfo{}
	if err := Unmarshal(&resp, block); err != nil {
		return nil, err
	}

	return block, nil
}

func GetWithdrawTransactionByHeight(height uint32, config *config.RpcConfig) ([]*base.WithdrawTxInfo, error) {
	resp, err := CallAndUnmarshal("getwithdrawtransactionsbyheight", Param("height", height), config)
	if err != nil {
		return nil, err
	}
	txs := make([]*base.WithdrawTxInfo, 0)
	if err = Unmarshal(&resp, &txs); err != nil {
		log.Error("[GetWithdrawTransactionByHeight] received invalid response")
		return nil, err
	}
	log.Debug("[GetWithdrawTransactionByHeight] len transactions:", len(txs), "transactions:", txs)

	return txs, nil
}

func GetIllegalEvidenceByHeight(height uint32, config *config.RpcConfig) ([]*base.SidechainIllegalDataInfo, error) {
	resp, err := CallAndUnmarshal("getillegalevidencebyheight", Param("height", height), config)
	if err != nil {
		return nil, err
	}
	evidences := make([]*base.SidechainIllegalDataInfo, 0)
	if err = Unmarshal(&resp, &evidences); err != nil {
		log.Error("[GetIllegalEvidenceByHeight] received invalid response")
		return nil, err
	}

	return evidences, nil
}

func CheckIllegalEvidence(evidence *base.SidechainIllegalDataInfo, config *config.RpcConfig) (bool, error) {
	param := map[string]interface{}{"evidence": evidence}
	resp, err := CallAndUnmarshal("checkillegalevidence", param, config)
	if err != nil {
		return false, err
	}
	result := false
	if err = Unmarshal(&resp, &result); err != nil {
		log.Error("[CheckIllegalEvidence] received invalid response")
		return false, err
	}

	return result, nil
}

func GetTransactionInfoByHash(transactionHash string, config *config.RpcConfig) (*base.WithdrawTxInfo, error) {
	hashBytes, err := common.HexStringToBytes(transactionHash)
	if err != nil {
		return nil, err
	}
	reversedHashBytes := common.BytesReverse(hashBytes)
	reversedHashStr := common.BytesToHexString(reversedHashBytes)

	result, err := CallAndUnmarshal("getwithdrawtransaction", Param("txid", reversedHashStr), config)
	if err != nil {
		return nil, err
	}

	tx := &base.WithdrawTxInfo{}
	if err := Unmarshal(&result, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func GetExistWithdrawTransactions(txs []string) ([]string, error) {
	parameter := make(map[string]interface{})
	parameter["txs"] = txs
	result, err := CallAndUnmarshal("getexistwithdrawtransactions",
		parameter, config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}

	var removeTxs []string
	if err := Unmarshal(&result, &removeTxs); err != nil {
		return nil, err
	}
	return removeTxs, nil
}

func GetExistDepositTransactions(txs []string, config *config.RpcConfig) ([]string, error) {
	parameter := make(map[string]interface{})
	parameter["txs"] = txs
	result, err := CallAndUnmarshal("getexistdeposittransactions", parameter, config)
	if err != nil {
		return nil, err
	}

	var removeTxs []string
	if err := Unmarshal(&result, &removeTxs); err != nil {
		return nil, err
	}
	return removeTxs, nil
}

func GetWithdrawUTXOsByAmount(genesisAddress string, amount common.Fixed64, config *config.RpcConfig) ([]base.UTXOInfo, error) {
	parameter := make(map[string]interface{})
	parameter["address"] = genesisAddress
	parameter["amount"] = amount.String()
	result, err := CallAndUnmarshal("getutxosbyamount", parameter, config)
	if err != nil {
		return nil, err
	}

	var utxoInfos []base.UTXOInfo
	if err := Unmarshal(&result, &utxoInfos); err != nil {
		return nil, err
	}

	return utxoInfos, nil
}

func GetAmountByInputs(inputs []*types.Input, config *config.RpcConfig) (common.Fixed64, error) {
	buf := new(bytes.Buffer)
	if err := common.WriteVarUint(buf, uint64(len(inputs))); err != nil {
		return 0, err
	}
	for _, input := range inputs {
		if err := input.Serialize(buf); err != nil {
			return 0, err
		}
	}
	parameter := make(map[string]interface{})
	parameter["inputs"] = common.BytesToHexString(buf.Bytes())
	result, err := CallAndUnmarshal("getamountbyinputs", parameter, config)
	if err != nil {
		return 0, err
	}
	if a, ok := result.(string); ok {
		amount, err := common.StringToFixed64(a)
		if err != nil {
			return 0, err
		}
		return *amount, nil
	}
	return 0, errors.New("get amount by inputs failed")
}

func GetUnspentUtxo(addresses []string, config *config.RpcConfig) ([]base.UTXOInfo, error) {
	parameter := make(map[string]interface{})
	parameter["addresses"] = addresses
	result, err := CallAndUnmarshal("listunspent", parameter, config)
	if err != nil {
		return nil, err
	}

	var utxoInfos []base.UTXOInfo
	if err := Unmarshal(&result, &utxoInfos); err != nil {
		return nil, err
	}
	return utxoInfos, nil
}

func post(url string, contentType string, user string, pass string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	auth := user + ":" + pass
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", basicAuth)
	req.Header.Set("Content-Type", contentType)

	client := *http.DefaultClient
	client.Timeout = time.Minute
	return client.Do(req)
}

func Call(method string, params map[string]interface{}, config *config.RpcConfig) ([]byte, error) {
	url := "http://" + config.IpAddress + ":" + strconv.Itoa(config.HttpJsonPort)
	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}

	resp, err := post(url, "application/json", config.User, config.Pass, strings.NewReader(string(data)))
	if err != nil {
		log.Debug("POST requset err:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func CallAndUnmarshal(method string, params map[string]interface{}, config *config.RpcConfig) (interface{}, error) {
	body, err := Call(method, params, config)
	if err != nil {
		return nil, err
	}

	resp := Response{}
	if err = json.Unmarshal(body, &resp); err != nil {
		return string(body), nil
	}

	if resp.Error != nil {
		return nil, errors.New(resp.Error.Message)
	}

	return resp.Result, nil
}

func CallAndUnmarshalResponse(method string, params map[string]interface{}, config *config.RpcConfig) (Response, error) {
	body, err := Call(method, params, config)
	if err != nil {
		return Response{}, err
	}

	resp := Response{}
	if err = json.Unmarshal(body, &resp); err != nil {
		return Response{}, err
	}

	return resp, nil
}

func Unmarshal(result interface{}, target interface{}) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, target); err != nil {
		return err
	}
	return nil
}
