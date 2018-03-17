package config

import (
	"bytes"
	"io/ioutil"
	"encoding/json"
	"fmt"
)

const (
	ConfigFilename = "./cli-config.json"
)

var config *Configuration // The single instance of config

type Configuration struct {
	IpAddress    string `json:IpAddress`
	HttpJsonPort int    `json:"HttpJsonPort"`
}

func (config *Configuration) readConfigFile() error {
	data, err := ioutil.ReadFile(ConfigFilename)
	if err != nil {
		return err
	}
	// Remove the UTF-8 Byte Order Mark
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	err = json.Unmarshal(data, config)
	if err != nil {
		return err
	}
	return nil
}

func Config() *Configuration {
	if config == nil {
		config = &Configuration{
			"localhost",
			20336,
		}
		err := config.readConfigFile()
		if err != nil {
			fmt.Println("Read config file error:", err)
		}
	}
	return config
}
