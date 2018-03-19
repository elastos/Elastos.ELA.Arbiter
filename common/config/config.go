package config

import (
	"io/ioutil"
	"os"
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	DefaultConfigFilename = "./config.json"
)

var (
	Parameters configParams
)

type Configuration struct {
	Version				int 				`json:"Version"`

	//arbitrator group
	MemberCount			int					`json:"MemberCount"`
	MainNode 			*MainNodeConfig 	`json:"MainNode"`
	SideNodeList		[]*SideNodeConfig	`json:"SideNodeList"`

	HttpJsonPort		int 				`json:"HttpJsonPort"`
	PrintLevel          int              	`json:"PrintLevel"`
	MaxLogSize          int64            	`json:"MaxLogSize"`
}

type RpcConfig struct {
	IpAddress    		string 				`json:"IpAddress"`
	HttpJsonPort 		int    				`json:"HttpJsonPort"`
}

type MainNodeConfig struct {
	Rpc					*RpcConfig			`json:"Rpc"`
}

type SideNodeConfig struct {
	Rpc					*RpcConfig			`json:"Rpc"`

	GenesisBlockAddress	string				`json:"GenesisBlockAddress"`
}

type ConfigFile struct {
	ConfigFile Configuration `json:"Configuration"`
}

type configParams struct {
	*Configuration
}

func init() {
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