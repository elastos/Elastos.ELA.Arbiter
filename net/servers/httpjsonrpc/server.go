package httpjsonrpc

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/errors"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	. "github.com/elastos/Elastos.ELA.Arbiter/net/servers"
)

//an instance of the multiplexer
var mainMux map[string]func(Params) map[string]interface{}

func StartRPCServer() {
	mainMux = make(map[string]func(Params) map[string]interface{})

	http.HandleFunc("/", Handle)

	mainMux["submitcomplain"] = SubmitComplain
	mainMux["getcomplainstatus"] = GetComplainStatus

	mainMux["getinfo"] = GetInfo
	mainMux["getsidemininginfo"] = GetSideMiningInfo
	mainMux["getmainchainblockheight"] = GetMainChainBlockHeight
	mainMux["getsidechainblockheight"] = GetSideChainBlockHeight
	mainMux["getfinisheddeposittxs"] = GetFinishedDepositTxs
	mainMux["getfinishedwithdrawtxs"] = GetFinishedWithdrawTxs
	mainMux["getgitversion"] = GetGitVersion
	mainMux["getspvheight"] = GetSPVHeight

	err := http.ListenAndServe(":"+strconv.Itoa(config.Parameters.HttpJsonPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

//this is the funciton that should be called in order to answer an rpc call
//should be registered like "http.AddMethod("/", httpjsonrpc.Handle)"
func Handle(w http.ResponseWriter, r *http.Request) {
	//JSON RPC commands should be POSTs
	if r.Method != "POST" {
		log.Warn("HTTP JSON RPC Handle - Method!=\"POST\"")
		return
	}

	//check if there is Request Body to read
	if r.Body == nil {
		log.Warn("HTTP JSON RPC Handle - Request body is nil")
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

func checkMethod(request map[string]interface{}) (func(Params) map[string]interface{}, interface{}, bool) {
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
