package rpc

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/dpos/p2p"
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

func GetActiveDposPeers() (result []p2p.PeerAddr, err error) {
	resp, err := CallAndUnmarshal("getactivedpospeers", nil, config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}

	peerAddrMap := make(map[string]string)
	Unmarshal(&resp, peerAddrMap)

	for k, v := range peerAddrMap {
		var id peer.PID
		pk, err := common.HexStringToBytes(k)
		if err != nil {
			return nil, err
		}

		copy(id[:], pk)
		result = append(result, p2p.PeerAddr{
			PID:  id,
			Addr: v,
		})
	}
	return result, nil
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
	result, err := CallAndUnmarshal("getblockcount", nil, config)
	if err != nil {
		return 0, err
	}
	if count, ok := result.(float64); ok {
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
	Unmarshal(&resp, block)

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
	Unmarshal(&result, tx)

	return tx, nil
}

func GetExistWithdrawTransactions(txs []string) ([]string, error) {
	infoBytes, err := json.Marshal(txs)
	if err != nil {
		return nil, err
	}

	result, err := CallAndUnmarshal("getexistwithdrawtransactions",
		Param("txs", common.BytesToHexString(infoBytes)), config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}

	var removeTxs []string
	Unmarshal(&result, &removeTxs)
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
	Unmarshal(&result, &removeTxs)
	return removeTxs, nil
}

func GetUnspentUtxo(addresses []string, config *config.RpcConfig) ([]base.UTXOInfo, error) {
	parameter := make(map[string]interface{})
	parameter["addresses"] = addresses
	result, err := CallAndUnmarshal("listunspent", parameter, config)
	if err != nil {
		return nil, err
	}

	var utxoInfos []base.UTXOInfo
	Unmarshal(&result, &utxoInfos)

	return utxoInfos, nil
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

	resp, err := http.Post(url, "application/json", strings.NewReader(string(data)))
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
	err = json.Unmarshal(body, &resp)
	if err != nil {
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
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return Response{}, err
	}

	return resp, nil
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
