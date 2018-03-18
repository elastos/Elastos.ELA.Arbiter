package base

import (
	"io/ioutil"
	"os"
	"Elastos.ELA.Arbiter/common/log"
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
	Version				int 			`json:"Version"`

	//arbitrator group
	MemberCount			int				`json:"MemberCount"`

	MainRpc 			*RpcConfig `json:"MainNode"`
}

type RpcConfig struct {
	IpAddress    string `json:IpAddress`
	HttpJsonPort int    `json:"HttpJsonPort"`
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
		log.Error(fmt.Sprintf("File error: %v\n", e))
		os.Exit(1)
	}
	// Remove the UTF-8 Byte Order Mark
	file = bytes.TrimPrefix(file, []byte("\xef\xbb\xbf"))

	config := ConfigFile{}
	e = json.Unmarshal(file, &config)
	if e != nil {
		log.Error(fmt.Sprintf("Unmarshal json file erro %v", e))
		os.Exit(1)
	}

	Parameters.Configuration = &(config.ConfigFile)
}