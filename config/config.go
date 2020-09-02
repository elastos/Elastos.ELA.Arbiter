package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"

	"github.com/elastos/Elastos.ELA/common"
	elacfg "github.com/elastos/Elastos.ELA/common/config"
)

const (
	DefaultConfigFilename = "./config.json"
)

var (
	Version    string
	Parameters configParams

	DataPath   = "elastos_arbiter"
	DataDir    = "data"
	SpvDir     = "spv"
	LogDir     = "logs"
	ArbiterDir = "arbiter"
)

type RpcConfiguration struct {
	User        string   `json:"User"`
	Pass        string   `json:"Pass"`
	WhiteIPList []string `json:"WhiteIPList"`
}

type Configuration struct {
	ActiveNet string `json:"ActiveNet"`
	Magic     uint32 `json:"Magic"`
	Version   uint32 `json:"Version"`
	NodePort  uint16 `json:"NodePort"`

	MainNode     *MainNodeConfig   `json:"MainNode"`
	SideNodeList []*SideNodeConfig `json:"SideNodeList"`

	SyncInterval  time.Duration `json:"SyncInterval"`
	HttpJsonPort  int           `json:"HttpJsonPort"`
	HttpRestPort  uint16        `json:"HttpRestPort"`
	PrintLevel    uint8         `json:"PrintLevel"`
	SPVPrintLevel uint8         `json:"SPVPrintLevel"`
	MaxLogsSize   int64         `json:"MaxLogsSize"`
	MaxPerLogSize int64         `json:"MaxPerLogSize"`

	SideChainMonitorScanInterval time.Duration    `json:"SideChainMonitorScanInterval"`
	ClearTransactionInterval     time.Duration    `json:"ClearTransactionInterval"`
	MinOutbound                  int              `json:"MinOutbound"`
	MaxConnections               int              `json:"MaxConnections"`
	SideAuxPowFee                int              `json:"SideAuxPowFee"`
	MinThreshold                 int              `json:"MinThreshold"`
	DepositAmount                int              `json:"DepositAmount"`
	CRCOnlyDPOSHeight            uint32           `json:"CRCOnlyDPOSHeight"`
	CRClaimDPOSNodeStartHeight   uint32           `json:"CRClaimDPOSNodeStartHeight"`
	MaxTxsPerWithdrawTx          int              `json:"MaxTxsPerWithdrawTx"`
	OriginCrossChainArbiters     []string         `json:"OriginCrossChainArbiters"`
	CRCCrossChainArbiters        []string         `json:"CRCCrossChainArbiters"`
	RpcConfiguration             RpcConfiguration `json:"RpcConfiguration"`
	DPoSNetAddress               string           `json:"DPoSNetAddress"`
}

type RpcConfig struct {
	IpAddress    string `json:"IpAddress"`
	HttpJsonPort int    `json:"HttpJsonPort"`
	User         string `json:"User"`
	Pass         string `json:"Pass"`
}

type MainNodeConfig struct {
	Rpc               *RpcConfig `json:"Rpc"`
	SpvSeedList       []string   `json:"SpvSeedList"`
	DefaultPort       uint16     `json:"DefaultPort"`
	Magic             uint32     `json:"Magic"`
	FoundationAddress string     `json:"FoundationAddress"`
}

type SideNodeConfig struct {
	Rpc *RpcConfig `json:"Rpc"`

	ExchangeRate        float64 `json:"ExchangeRate"`
	GenesisBlockAddress string  `json:"GenesisBlockAddress"`
	GenesisBlock        string  `json:"GenesisBlock"`
	KeystoreFile        string  `json:"KeystoreFile"`
	MiningAddr          string  `json:"MiningAddr"`
	PayToAddr           string  `json:"PayToAddr"`
	PowChain            bool    `json:"PowChain"`
}

type ConfigFile struct {
	ConfigFile Configuration `json:"Configuration"`
}

type configParams struct {
	*Configuration
}

func GetRpcConfig(genesisBlockHash string) (*RpcConfig, bool) {
	for _, node := range Parameters.SideNodeList {
		if node.GenesisBlock == genesisBlockHash {
			return node.Rpc, true
		}
	}
	return nil, false
}

func GetSpvChainParams() *elacfg.Params {
	var params *elacfg.Params
	switch strings.ToLower(Parameters.ActiveNet) {
	case "testnet", "test":
		params = elacfg.DefaultParams.TestNet()

	case "regnet", "reg":
		params = elacfg.DefaultParams.RegNet()

	default:
		params = &elacfg.DefaultParams
	}

	mncfg := Parameters.MainNode
	if mncfg.Magic != 0 {
		params.Magic = mncfg.Magic
	}
	if mncfg.FoundationAddress != "" {
		address, err := common.Uint168FromAddress(mncfg.FoundationAddress)
		if err != nil {
			fmt.Printf("invalid foundation address")
			os.Exit(1)
		}
		params.Foundation = *address
		params.GenesisBlock = elacfg.GenesisBlock(address)
	}
	if mncfg.DefaultPort != 0 {
		params.DefaultPort = mncfg.DefaultPort
	}
	params.DNSSeeds = nil
	return params
}

func init() {
	file, e := ioutil.ReadFile(DefaultConfigFilename)
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		return
	}
	i := ConfigFile{}
	// Remove the UTF-8 Byte Order Mark
	file = bytes.TrimPrefix(file, []byte("\xef\xbb\xbf"))

	e = json.Unmarshal(file, &i)
	var config ConfigFile
	switch strings.ToLower(i.ConfigFile.ActiveNet) {
	case "testnet", "test":
		config = ConfigFile{
			ConfigFile: Configuration{
				Magic:                        0,
				Version:                      0,
				NodePort:                     21538,
				HttpJsonPort:                 21536,
				PrintLevel:                   1,
				SPVPrintLevel:                1,
				MaxLogsSize:                  500,
				SyncInterval:                 1000,
				SideChainMonitorScanInterval: 1000,
				ClearTransactionInterval:     60000,
				MinOutbound:                  3,
				MaxConnections:               8,
				SideAuxPowFee:                50000,
				MinThreshold:                 1000000,
				DepositAmount:                1000000,
				MaxTxsPerWithdrawTx:          1000,
				RpcConfiguration: RpcConfiguration{
					User:        "",
					Pass:        "",
					WhiteIPList: []string{"127.0.0.1"},
				},
				DPoSNetAddress:    "127.0.0.1:21339",
				CRCOnlyDPOSHeight: 211000,
				CRCCrossChainArbiters: []string{
					"03e435ccd6073813917c2d841a0815d21301ec3286bc1412bb5b099178c68a10b6",
					"038a1829b4b2bee784a99bebabbfecfec53f33dadeeeff21b460f8b4fc7c2ca771",
					"02435df9a4728e6250283cfa8215f16b48948d71936c4600b3a5b1c6fde70503ae",
					"027d44ee7e7a6c6ff13a130d15b18c75a3b47494c3e54fcffe5f4b10e225351e09",
					"02ad972fbfce4aaa797425138e4f3b22bcfa765ffad88b8a5af0ab515161c0a365",
					"0373eeae2bac0f5f14373ca603fe2c9caa9c7a79c7793246cec415d005e2fe53c0",
					"03503011cc4e44b94f73ed2c76c73182a75b4863f23d1e7083025eead945a8e764",
					"0270b6880e7fab8d02bea7d22639d7b5e07279dd6477baa713dacf99bb1d65de69",
					"030eed9f9c1d70307beba52ddb72a24a02582c0ee626ec93ee1dcef2eb308852dd",
					"026bba43feb19ce5859ffcf0ce9dd8b9d625130b686221da8b445fa9b8f978d7b9",
					"02bf9e37b3db0cbe86acf76a76578c6b17b4146df101ec934a00045f7d201f06dd",
					"03111f1247c66755d369a8c8b3a736dfd5cf464ca6735b659533cbe1268cd102a9",
				},
				OriginCrossChainArbiters: []string{
					"03e333657c788a20577c0288559bd489ee65514748d18cb1dc7560ae4ce3d45613",
					"02dd22722c3b3a284929e4859b07e6a706595066ddd2a0b38e5837403718fb047c",
					"03e4473b918b499e4112d281d805fc8d8ae7ac0a71ff938cba78006bf12dd90a85",
					"03dd66833d28bac530ca80af0efbfc2ec43b4b87504a41ab4946702254e7f48961",
					"02c8a87c076112a1b344633184673cfb0bb6bce1aca28c78986a7b1047d257a448",
				},
				CRClaimDPOSNodeStartHeight: 1000000, // TODO reset it later
			},
		}

	case "regnet", "reg":
		config = ConfigFile{
			ConfigFile: Configuration{
				Magic:                        0,
				Version:                      0,
				NodePort:                     22538,
				HttpJsonPort:                 22536,
				PrintLevel:                   0,
				SPVPrintLevel:                0,
				MaxLogsSize:                  500,
				SyncInterval:                 1000,
				SideChainMonitorScanInterval: 1000,
				ClearTransactionInterval:     60000,
				MinOutbound:                  3,
				MaxConnections:               8,
				SideAuxPowFee:                50000,
				MinThreshold:                 1000000,
				DepositAmount:                1000000,
				MaxTxsPerWithdrawTx:          1000,
				RpcConfiguration: RpcConfiguration{
					User:        "",
					Pass:        "",
					WhiteIPList: []string{"127.0.0.1"},
				},
				DPoSNetAddress:    "127.0.0.1:22339",
				CRCOnlyDPOSHeight: 211000,
				CRCCrossChainArbiters: []string{
					"0306e3deefee78e0e25f88e98f1f3290ccea98f08dd3a890616755f1a066c4b9b8",
					"02b56a669d713db863c60171001a2eb155679cad186e9542486b93fa31ace78303",
					"0250c5019a00f8bb4fd59bb6d613c70a39bb3026b87cfa247fd26f59fd04987855",
					"02e00112e3e9defe0f38f33aaa55551c8fcad6aea79ab2b0f1ec41517fdd05950a",
					"020aa2d111866b59c70c5acc60110ef81208dcdc6f17f570e90d5c65b83349134f",
					"03cd41a8ed6104c1170332b02810237713369d0934282ca9885948960ae483a06d",
					"02939f638f3923e6d990a70a2126590d5b31a825a0f506958b99e0a42b731670ca",
					"032ade27506951c25127b0d2cb61d164e0bad8aec3f9c2e6785725a6ab6f4ad493",
					"03f716b21d7ae9c62789a5d48aefb16ba1e797b04a2ec1424cd6d3e2e0b43db8cb",
					"03488b0aace5fe5ee5a1564555819074b96cee1db5e7be1d74625240ef82ddd295",
					"03c559769d5f7bb64c28f11760cb36a2933596ca8a966bc36a09d50c24c48cc3e8",
					"03b5d90257ad24caf22fa8a11ce270ea57f3c2597e52322b453d4919ebec4e6300",
				},
				OriginCrossChainArbiters: []string{
					"03e333657c788a20577c0288559bd489ee65514748d18cb1dc7560ae4ce3d45613",
					"02dd22722c3b3a284929e4859b07e6a706595066ddd2a0b38e5837403718fb047c",
					"03e4473b918b499e4112d281d805fc8d8ae7ac0a71ff938cba78006bf12dd90a85",
					"03dd66833d28bac530ca80af0efbfc2ec43b4b87504a41ab4946702254e7f48961",
					"02c8a87c076112a1b344633184673cfb0bb6bce1aca28c78986a7b1047d257a448",
				},
				CRClaimDPOSNodeStartHeight: 1000000, // TODO reset it later
			},
		}

	default:
		config = ConfigFile{
			ConfigFile: Configuration{
				Magic:                        0,
				Version:                      0,
				NodePort:                     20538,
				HttpJsonPort:                 20536,
				PrintLevel:                   1,
				SPVPrintLevel:                1,
				MaxLogsSize:                  500,
				SyncInterval:                 1000,
				SideChainMonitorScanInterval: 1000,
				ClearTransactionInterval:     60000,
				MinOutbound:                  3,
				MaxConnections:               8,
				SideAuxPowFee:                50000,
				MinThreshold:                 10000000,
				DepositAmount:                10000000,
				MaxTxsPerWithdrawTx:          1000,
				RpcConfiguration: RpcConfiguration{
					User:        "",
					Pass:        "",
					WhiteIPList: []string{"127.0.0.1"},
				},
				DPoSNetAddress:    "127.0.0.1:20339",
				CRCOnlyDPOSHeight: 343400,
				CRCCrossChainArbiters: []string{
					"02089d7e878171240ce0e3633d3ddc8b1128bc221f6b5f0d1551caa717c7493062",
					"0268214956b8421c0621d62cf2f0b20a02c2dc8c2cc89528aff9bd43b45ed34b9f",
					"03cce325c55057d2c8e3fb03fb5871794e73b85821e8d0f96a7e4510b4a922fad5",
					"02661637ae97c3af0580e1954ee80a7323973b256ca862cfcf01b4a18432670db4",
					"027d816821705e425415eb64a9704f25b4cd7eaca79616b0881fc92ac44ff8a46b",
					"02d4a8f5016ae22b1acdf8a2d72f6eb712932213804efd2ce30ca8d0b9b4295ac5",
					"029a4d8e4c99a1199f67a25d79724e14f8e6992a0c8b8acf102682bd8f500ce0c1",
					"02871b650700137defc5d34a11e56a4187f43e74bb078e147dd4048b8f3c81209f",
					"02fc66cba365f9957bcb2030e89a57fb3019c57ea057978756c1d46d40dfdd4df0",
					"03e3fe6124a4ea269224f5f43552250d627b4133cfd49d1f9e0283d0cd2fd209bc",
					"02b95b000f087a97e988c24331bf6769b4a75e4b7d5d2a38105092a3aa841be33b",
					"02a0aa9eac0e168f3474c2a0d04e50130833905740a5270e8a44d6c6e85cf6d98c",
				},
				OriginCrossChainArbiters: []string{
					"0248df6705a909432be041e0baa25b8f648741018f70d1911f2ed28778db4b8fe4",
					"02771faf0f4d4235744b30972d5f2c470993920846c761e4d08889ecfdc061cddf",
					"0342196610e57d75ba3afa26e030092020aec56822104e465cba1d8f69f8d83c8e",
					"02fa3e0d14e0e93ca41c3c0f008679e417cf2adb6375dd4bbbee9ed8e8db606a56",
					"03ab3ecd1148b018d480224520917c6c3663a3631f198e3b25cf4c9c76786b7850",
				},
				CRClaimDPOSNodeStartHeight: 1000000, // TODO reset it later
			},
		}
	}

	Parameters.Configuration = &(config.ConfigFile)

	e = json.Unmarshal(file, &config)
	if e != nil {
		fmt.Printf("Unmarshal json file erro %v", e)
		os.Exit(1)
	}

	for _, side := range config.ConfigFile.SideNodeList {
		side.PowChain = true
	}

	e = json.Unmarshal(file, &config)
	if e != nil {
		fmt.Printf("Unmarshal json file erro %v", e)
		os.Exit(1)
	}

	//Parameters.Configuration = &(config.ConfigFile)

	var out bytes.Buffer
	err := json.Indent(&out, file, "", "")
	if err != nil {
		fmt.Printf("Config file error: %v\n", e)
		os.Exit(1)
	}

	if Parameters.Configuration.MainNode == nil {
		fmt.Printf("Need to set main node in config file\n")
		return
	}

	if Parameters.Configuration.SideNodeList == nil {
		fmt.Printf("Need to set side node list in config file\n")
		return
	}

	for _, node := range Parameters.SideNodeList {
		genesisBytes, err := common.HexStringToBytes(node.GenesisBlock)
		if err != nil {
			fmt.Printf("Side node genesis block hash error: %v\n", e)
			return
		}
		reversedGenesisBytes := common.BytesReverse(genesisBytes)
		reversedGenesisStr := common.BytesToHexString(reversedGenesisBytes)
		genesisBlockHash, err := common.Uint256FromHexString(reversedGenesisStr)
		if err != nil {
			fmt.Printf("Side node genesis block hash reverse error: %v\n", e)
			return
		}
		address, err := base.GetGenesisAddress(*genesisBlockHash)
		if err != nil {
			fmt.Printf("Side node genesis block hash to address error: %v\n", e)
			return
		}
		node.GenesisBlockAddress = address
		node.GenesisBlock = reversedGenesisStr
	}
}
