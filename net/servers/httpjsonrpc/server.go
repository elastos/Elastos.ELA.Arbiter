package httpjsonrpc

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/errors"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/net/servers"
)

//an instance of the multiplexer
var mainMux map[string]func(servers.Params) map[string]interface{}

func StartRPCServer(pServer *http.Server) {
	mainMux = make(map[string]func(servers.Params) map[string]interface{})

	mainMux["submitcomplain"] = servers.SubmitComplain
	mainMux["getcomplainstatus"] = servers.GetComplainStatus
	mainMux["getinfo"] = servers.GetInfo
	mainMux["getsidemininginfo"] = servers.GetSideMiningInfo
	mainMux["getmainchainblockheight"] = servers.GetMainChainBlockHeight
	mainMux["getsidechainblockheight"] = servers.GetSideChainBlockHeight
	mainMux["getfinisheddeposittxs"] = servers.GetFinishedDepositTxs
	mainMux["getfinishedwithdrawtxs"] = servers.GetFinishedWithdrawTxs
	mainMux["getgitversion"] = servers.GetGitVersion
	mainMux["getspvheight"] = servers.GetSPVHeight
	mainMux["getarbiterpeersinfo"] = servers.GetArbiterPeersInfo
	mainMux["setregistersidechainrpcinfo"] = servers.SetRegisterSideChainRPCInfo

	rpcServeMux := http.NewServeMux()
	rpcServeMux.HandleFunc("/", Handle)
	if pServer == nil {
		pServer = &http.Server{}
	}
	if pServer != nil {
		pServer.Handler = rpcServeMux
		pServer.ReadTimeout = 15 * time.Second
		pServer.WriteTimeout = 15 * time.Second
	}

	listerner, err := net.Listen("tcp4", ":"+strconv.Itoa(config.Parameters.HttpJsonPort))
	if err != nil {
		log.Fatal("Listen error: ", err.Error())
		return
	}
	err = pServer.Serve(listerner)
	if err != nil {
		log.Warnf("StartRPCServer : %v", err.Error())
	}
}

func Stop(s *http.Server) error {

	if s != nil {
		return s.Shutdown(context.Background())
	}
	return fmt.Errorf("server not started")
}

//this is the funciton that should be called in order to answer an rpc call
//should be registered like "http.AddMethod("/", httpjsonrpc.Handle)"
func Handle(w http.ResponseWriter, r *http.Request) {
	isClientAllowed := clientAllowed(r)
	if !isClientAllowed {
		log.Warn("HTTP Client ip is not allowd")
		http.Error(w, "Client ip is not allowd", http.StatusForbidden)
		return
	}
	//JSON RPC commands should be POSTs
	if r.Method != "POST" {
		log.Warn("HTTP JSON RPC Handle - Method!=\"POST\"")
		http.Error(w, "JSON RPC protocol only allows POST method", http.StatusMethodNotAllowed)
		return
	}

	//check if there is Request Body to read
	if r.Body == nil {
		log.Warn("HTTP JSON RPC Handle - Request body is nil")
		return
	}

	isCheckAuthOk := checkAuth(r)
	if !isCheckAuthOk {
		//log.Warn("client authenticate failed")
		http.Error(w, "client authenticate failed", http.StatusUnauthorized)
		return
	}

	//read the body of the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("HTTP JSON RPC Handle - ioutil.ReadAll: ", err)
		return
	}
	request := make(map[string]interface{})
	err = json.Unmarshal(body, &request)
	if err != nil {
		log.Error("HTTP JSON RPC Handle - json.Unmarshal: ", err)
		return
	}

	//get the corresponding function
	function, method, ok := checkMethod(request)

	if !ok {
		Error(w, errors.InvalidMethod, method)
		return
	}

	params, ok := checkParams(request)
	if !ok {
		Error(w, errors.InvalidParams, method)
		return
	}

	response := function(params)
	var data []byte
	if response["Error"] != errors.ErrCode(0) {
		data, _ = json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"result":  nil,
			"error": map[string]interface{}{
				"code":    response["Error"],
				"message": response["Result"],
				"id":      request["id"],
			},
		})

	} else {
		data, _ = json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"result":  response["Result"],
			"id":      request["id"],
			"error":   nil,
		})
	}
	w.Header().Set("Content-type", "application/json")
	w.Write(data)
}

func checkAuth(r *http.Request) bool {
	tempRpcConf := config.Parameters.RpcConfiguration
	if (tempRpcConf.User == tempRpcConf.Pass) && (len(tempRpcConf.User) == 0) {
		return true
	}
	authHeader := r.Header["Authorization"]
	if len(authHeader) <= 0 {
		return false
	}

	authSha256 := sha256.Sum256([]byte(authHeader[0]))

	login := tempRpcConf.User + ":" + tempRpcConf.Pass
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
	cfgAuthSha256 := sha256.Sum256([]byte(auth))

	resultCmp := subtle.ConstantTimeCompare(authSha256[:], cfgAuthSha256[:])
	if resultCmp == 1 {
		return true
	}

	// Request's auth doesn't match  user
	return false
}

func clientAllowed(r *http.Request) bool {
	log.Debugf("clientAllowed RpcConfiguration %v", config.Parameters.RpcConfiguration)
	//this ipAbbr  may be  ::1 when request is localhost
	ipAbbr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Errorf("RemoteAddr clientAllowed SplitHostPort failure %s \n", r.RemoteAddr)
		return false

	}
	//after ParseIP ::1 chg to 0:0:0:0:0:0:0:1 the true ip
	remoteIp := net.ParseIP(ipAbbr)

	if remoteIp == nil {
		log.Errorf("clientAllowed ParseIP ipAbbr %s failure  \n", ipAbbr)
		return false
	}

	if remoteIp.IsLoopback() {
		//log.Debugf("remoteIp %s IsLoopback\n", remoteIp)
		return true
	}

	for _, cfgIp := range config.Parameters.RpcConfiguration.WhiteIPList {
		//WhiteIPList have 0.0.0.0  allow all ip in
		if cfgIp == "0.0.0.0" {
			return true
		}
		if cfgIp == remoteIp.String() {
			return true
		}

	}
	return false
}

func checkMethod(request map[string]interface{}) (func(servers.Params) map[string]interface{}, interface{}, bool) {
	method := request["method"]
	if method == nil {
		return nil, method, false
	}
	switch method.(type) {
	case string:
	default:
		return nil, method, false
	}
	function, ok := mainMux[request["method"].(string)]
	if !ok {
		return nil, method, false
	}
	return function, nil, true
}

func checkParams(request map[string]interface{}) (map[string]interface{}, bool) {
	params := request["params"]
	if params == nil {
		return map[string]interface{}{}, true
	}
	switch params.(type) {
	case map[string]interface{}:
		return params.(map[string]interface{}), true
	default:
		return nil, false
	}
	return nil, false
}

func Error(w http.ResponseWriter, code errors.ErrCode, method interface{}) {
	//if the function does not exist
	log.Warn("HTTP JSON RPC Handle - No function to call for ", method)
	data, _ := json.Marshal(map[string]interface{}{
		"jsonpc": "2.0",
		"code":   code,
		"result": code.Message(),
	})
	w.Write(data)
}
