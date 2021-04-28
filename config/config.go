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
	// DefaultConfigFilename indicates the file name of config.
	DefaultConfigFilename = "./config.json"

	// NodePrefix indicates the prefix of node version.
	NodePrefix = "arbiter-"
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
	NewP2PProtocolVersionHeight  uint64           `json:"NewP2PProtocolVersionHeight"`
	DPOSNodeCrossChainHeight     uint32           `json:"DPOSNodeCrossChainHeight"`
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
	SyncStartHeight     uint32  `json:"SyncStartHeight"`
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
	if Parameters.CRClaimDPOSNodeStartHeight > 0 {
		params.CRClaimDPOSNodeStartHeight = Parameters.CRClaimDPOSNodeStartHeight
	}
	if Parameters.NewP2PProtocolVersionHeight > 0 {
		params.NewP2PProtocolVersionHeight = Parameters.NewP2PProtocolVersionHeight
	}
	if Parameters.DPOSNodeCrossChainHeight > 0 {
		params.DPOSNodeCrossChainHeight = Parameters.DPOSNodeCrossChainHeight
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
		config = testnet
	case "regnet", "reg":
		config = regnet
	default:
		config = mainnet
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
