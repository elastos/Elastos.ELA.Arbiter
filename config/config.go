package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

const (
	DefaultConfigFilename = "./config.json"
)

var (
	Parameters configParams
)

type Configuration struct {
	Magic    uint32   `json:"Magic"`
	Version  int      `json:"Version"`
	SeedList []string `json:"SeedList"`
	NodePort uint16   `json:"NodePort"`

	MainNode     *MainNodeConfig   `json:"MainNode"`
	SideNodeList []*SideNodeConfig `json:"SideNodeList"`

	SyncInterval time.Duration `json:"SyncInterval"`
	HttpJsonPort int           `json:"HttpJsonPort"`
	PrintLevel   int           `json:"PrintLevel"`
	MaxLogSize   int64         `json:"MaxLogSize"`

	SideChainMonitorScanInterval time.Duration `json:"SideChainMonitorScanInterval"`
}

type RpcConfig struct {
	IpAddress    string `json:"IpAddress"`
	HttpJsonPort int    `json:"HttpJsonPort"`
}

type MainNodeConfig struct {
	Rpc         *RpcConfig `json:"Rpc"`
	SpvSeedList []string   `json:SpvSeedList`
}

type SideNodeConfig struct {
	Rpc *RpcConfig `json:"Rpc"`

	Rate                float32 `json:"Rate"`
	GenesisBlockAddress string  `json:"GenesisBlockAddress"`
	GenesisBlock        string  `json:"GenesisBlock"`
}

type ConfigFile struct {
	ConfigFile Configuration `json:"Configuration"`
}

type configParams struct {
	*Configuration
}

func GetRpcConfig(genesisBlockHash string) (*RpcConfig, bool) {
	for _, node := range Parameters.SideNodeList {
		if node.GenesisBlockAddress == genesisBlockHash {
			return node.Rpc, true
		}
	}
	return nil, false
}

func Init() {
	file, e := ioutil.ReadFile(DefaultConfigFilename)
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}

	// Remove the UTF-8 Byte Order Mark
	file = bytes.TrimPrefix(file, []byte("\xef\xbb\xbf"))

	config := ConfigFile{}
	e = json.Unmarshal(file, &config)
	if e != nil {
		fmt.Printf("Unmarshal json file erro %v", e)
		os.Exit(1)
	}

	Parameters.Configuration = &(config.ConfigFile)
}
