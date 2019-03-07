package httpjsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA/utils/test"
)

//if bRunServer is true .run server for every testcase
var bRunServer bool = true

var (
	urlNotLoopBack string
	urlLoopBack    string
	urlLocalhost   string
	req_new        *bytes.Buffer
	clientAuthUser string
	clientAuthPass string
	pServer        *http.Server
)

func initUrl() {
	ipNotLoopBack := resolveHostIp()
	if ipNotLoopBack == "" {
		//t.Error("expecting not found get %", resp.Status)
		fmt.Printf("ipNotLoopBack  error should exit!!!!!!!!!!!!!!!")
		return
	}
	httpPrefix := "http://"
	httPostfix := ":20336/getmainchainblockheight"

	urlNotLoopBack = httpPrefix + ipNotLoopBack + httPostfix
	fmt.Printf("Before Test init url %v", urlNotLoopBack)

	urlLoopBack = "http://127.0.0.1:20336"
	urlLocalhost = "http://localhost:20336"
}
func isRunServer() bool {
	return bRunServer
}
func init() {

	log.Init("./Elastos", 1, 0, 0)
	initUrl()
	initReqObject()
}

func InitNewServer(conf config.RpcConfiguration) {
	pServer = new(http.Server)
	InitConf(conf)
}
func initReqObject() {

	type ReqObj struct {
		method string
	}
	var reqObj ReqObj
	reqStr := `{
	"method":"getinfo"
	
	}`
	if err := json.Unmarshal([]byte(reqStr), &reqObj); err == nil {
		//fmt.Printf("reqObj %+v\n", reqObj)
	} else {
		//fmt.Println(err)
	}
	///////////////
	req_new = bytes.NewBuffer([]byte(reqStr))
}

/*
hope： if no init RpcConfiguration. ip only accept IsLoopback  localhost
testcase1 : RpcConfiguration no init client no authorization
testcase1 : RpcConfiguration no init client have authorization

testcase2 : RpcConfiguration have User and Pass no ip.  User and Pass are correct
testcase2 : RpcConfiguration have User and Pass no ip.  User and Pass are wrong
testcase2 : RpcConfiguration have User and Pass no ip.  User is wrong
testcase2 : RpcConfiguration have User and Pass no ip.  Pass is  wrong
testcase2 : RpcConfiguration have User and Pass no ip.  client no authorization


testcase3 : RpcConfiguration have ip no User and Pass. Ip is allowed
testcase3 : RpcConfiguration have ip no User and Pass. Ip is forbid    *

testcase4 : RpcConfiguration have ip  User and Pass. All  is correct
testcase4 : RpcConfiguration have ip  User and Pass. ip  is wrong      *
testcase4 : RpcConfiguration have ip  User and Pass. User  is wrong
testcase4 : RpcConfiguration have ip  User and Pass. Pass  is wrong

testcase5 : whiteiplist 0.0.0.0  allow all ip
*/

func GetInternalIP() string {
	itf, _ := net.InterfaceByName("en0") //here your interface
	item, _ := itf.Addrs()
	var ip net.IP
	for _, addr := range item {
		switch v := addr.(type) {
		case *net.IPNet:
			if !v.IP.IsLoopback() {
				if v.IP.To4() != nil { //Verify if IP is IPV4
					ip = v.IP
				}
			}
		}
	}
	if ip != nil {
		return ip.String()
	} else {
		return ""
	}
}
func resolveHostIp() string {

	netInterfaceAddresses, err := net.InterfaceAddrs()

	if err != nil {
		return ""
	}

	for _, netInterfaceAddress := range netInterfaceAddresses {

		networkIp, ok := netInterfaceAddress.(*net.IPNet)

		if ok && !networkIp.IP.IsLoopback() && networkIp.IP.To4() != nil {

			ip := networkIp.IP.String()

			fmt.Println("Resolved Host IP: " + ip)

			return ip
		}
	}
	return ""
}

func PostReq(url string, withAuthorization bool, expectStatus int, t *testing.T) {
	//t.Logf("PostReq req_new !!!!!!!!! %v", req_new)
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, req_new)
	if err != nil {
	}
	req.Header.Set("Content-Type", "application/json")
	if withAuthorization {
		req.SetBasicAuth(clientAuthUser, clientAuthPass)
	}
	resp, _ := client.Do(req)
	if resp != nil {
		if resp.StatusCode != expectStatus {
			t.Error("expecting not found get resp.Status %", resp.Status)
		}
	}
}
func Wait() {

	select {
	case <-time.After(time.Second * 3):
		Stop(pServer)
	}
}
func InitConf(conf config.RpcConfiguration) {
	config.Parameters.RpcConfiguration = conf
}
func TestServer_NotInitRpcConf(t *testing.T) {

	t.Logf("NotInitRpcConf1 request with no authorization and 127.0.0.1 begin")
	test.SkipShort(t)

	//modify config RpcConfiguration
	svrConf := config.RpcConfiguration{
		User:        "",
		Pass:        "",
		WhiteIPList: []string{""},
	}
	InitNewServer(svrConf)
	if isRunServer() {
		go StartRPCServer(pServer)
	}

	urlLoopBackNoAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_NotInitRpcConf    urlLoopBackNoAuthTest end")

	}
	urlLoopBackNoAuthTest(urlLoopBack, false, http.StatusOK, t)

	urlLoopBackWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_NotInitRpcConf    urlLoopBackWithAuthTest end")

	}
	urlLoopBackWithAuthTest(urlLoopBack, true, http.StatusOK, t)

	urlLocalhostWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_NotInitRpcConf    urlLocalhostWithAuthTest end")

	}
	urlLocalhostWithAuthTest(urlLocalhost, true, http.StatusOK, t)

	urlNotLoopBackWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_NotInitRpcConf    urlNotLoopBackWithAuthTest end")

	}
	urlNotLoopBackWithAuthTest(urlNotLoopBack, true, http.StatusForbidden, t)

	Wait()
}

//
func TestServer_WithUserPassNoIp(t *testing.T) {
	t.Logf("WithUserPassNoIp1    authorization(user,pass) ok and localhost begin")
	test.SkipShort(t)

	svrConf := config.RpcConfiguration{
		User:        "ElaUser",
		Pass:        "Ela123",
		WhiteIPList: []string{""},
	}
	InitNewServer(svrConf)

	if isRunServer() {
		go StartRPCServer(pServer)
	}

	urlLocalhostWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithUserPassNoIp    urlLocalhostWithAuthTest end")

	}
	urlLocalhostWithAuthTest(urlLocalhost, true, http.StatusOK, t)

	//////////////////////////
	urlLoopBackWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithUserPassNoIp    urlLoopBackWithAuthTest end")

	}
	urlLoopBackWithAuthTest(urlLoopBack, true, http.StatusOK, t)

	urlNotLoopBackWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithUserPassNoIp    urlNotLoopBackWithAuthTest end")

	}
	urlNotLoopBackWithAuthTest(urlLoopBack, true, http.StatusOK, t)
	////////////////////////

	urlLocalhostWithAuthWrongUserPassTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = "1111"
		clientAuthPass = "1111"
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithUserPassNoIp    urlLocalhostWithAuthWrongUserPassTest end")

	}
	urlLocalhostWithAuthWrongUserPassTest(urlLocalhost, true, http.StatusUnauthorized, t)

	urlLocalhostWithAuthWrongUserTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = "1111"
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithUserPassNoIp    urlLocalhostWithAuthWrongUserTest end")

	}

	urlLocalhostWithAuthWrongUserTest(urlLocalhost, true, http.StatusUnauthorized, t)

	urlLocalhostWithAuthWrongPassTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = "123"
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithUserPassNoIp    urlLocalhostWithAuthWrongPassTest end")

	}
	urlLocalhostWithAuthWrongPassTest(urlLocalhost, true, http.StatusUnauthorized, t)

	urlLocalhostNoAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = "123"
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithUserPassNoIp    urlLocalhostNoAuthTest end")

	}
	urlLocalhostNoAuthTest(urlLocalhost, false, http.StatusUnauthorized, t)

	Wait()
}

func TestServer_NoUserPassWithIp(t *testing.T) {

	t.Logf("NoUserPassWithIp1  no  user and pass and whiteiplist is allowd")

	test.SkipShort(t)
	svrConf := config.RpcConfiguration{
		//User:        "ElaUser",
		//Pass:        "Ela123",
		WhiteIPList: []string{"127.0.0.1"},
	}
	InitNewServer(svrConf)

	if isRunServer() {
		go StartRPCServer(pServer)
	}

	urlLocalhostNoAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_NoUserPassWithIp    urlLocalhostNoAuthTest end")

	}
	urlLocalhostNoAuthTest(urlLocalhost, false, http.StatusOK, t)

	urlLoopBackNoAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_NoUserPassWithIp    urlLoopBackNoAuthTest end")

	}
	urlLoopBackNoAuthTest(urlLoopBack, false, http.StatusOK, t)

	urlNotLoopBacktNoAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_NoUserPassWithIp    urlNotLoopBacktNoAuthTest end")

	}
	urlNotLoopBacktNoAuthTest(urlNotLoopBack, false, http.StatusForbidden, t)

	urlLoopBackWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_NoUserPassWithIp    urlLoopBackWithAuthTest end")

	}
	urlLoopBackWithAuthTest(urlLoopBack, true, http.StatusOK, t)

	Wait()
}

func TestServer_WithUserPassWithIp(t *testing.T) {

	t.Logf("WithUserPassWithIp1  with  user and pass and whiteiplist are all correct")
	initReqObject()

	test.SkipShort(t)
	svrConf := config.RpcConfiguration{
		User:        "ElaUser",
		Pass:        "Ela123",
		WhiteIPList: []string{"127.0.0.1"},
	}
	InitNewServer(svrConf)

	if isRunServer() {
		go StartRPCServer(pServer)
	}

	urlLoopbackWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithUserPassWithIp    urlLoopbackWithAuthTest end")

	}
	urlLoopbackWithAuthTest(urlLoopBack, true, http.StatusOK, t)

	urlNotLoopbackWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithUserPassWithIp    urlNotLoopbackWithAuthTest end")

	}
	urlNotLoopbackWithAuthTest(urlNotLoopBack, true, http.StatusForbidden, t)

	urlLoopbackWithAuthWrongUserTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = "1111"
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithUserPassWithIp    urlLoopbackWithAuthWrongUserTest end")

	}
	urlLoopbackWithAuthWrongUserTest(urlLoopBack, true, http.StatusUnauthorized, t)

	urlLoopbackWithAuthWrongPassTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = "1111"
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithUserPassWithIp    urlLoopbackWithAuthWrongPassTest end")

	}
	urlLoopbackWithAuthWrongPassTest(urlLoopBack, true, http.StatusUnauthorized, t)

	Wait()
}

func TestServer_WithIp0000(t *testing.T) {

	t.Logf("WithIp0000  with  user and pass and ip 0.0.0.0. client user 192.168 expect ok")

	test.SkipShort(t)
	svrConf := config.RpcConfiguration{
		User:        "ElaUser",
		Pass:        "Ela123",
		WhiteIPList: []string{"0.0.0.0"},
	}
	InitNewServer(svrConf)

	if isRunServer() {
		go StartRPCServer(pServer)
	}
	urlNotLoopbackWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithIp0000    urlNotLoopbackWithAuthTest end")

	}
	urlNotLoopbackWithAuthTest(urlNotLoopBack, true, http.StatusOK, t)

	urlLoopbackWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithIp0000    urlLoopbackWithAuthTest end")

	}
	urlLoopbackWithAuthTest(urlLoopBack, true, http.StatusOK, t)

	urlLocalhostWithAuthTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = svrConf.User
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithIp0000    urlLocalhostWithAuthTest end")

	}
	urlLocalhostWithAuthTest(urlLocalhost, true, http.StatusOK, t)

	urlLoopbackWithAuthWrongUserTest := func(url string, withAuthorization bool, expectStatus int, t *testing.T) {
		clientAuthUser = "111"
		clientAuthPass = svrConf.Pass
		initReqObject()
		PostReq(url, withAuthorization, expectStatus, t)
		t.Logf("TestServer_WithIp0000    urlLoopbackWithAuthWrongUserTest end")

	}
	urlLoopbackWithAuthWrongUserTest(urlLoopBack, true, http.StatusUnauthorized, t)

	Wait()
}
