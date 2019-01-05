package config

import (
	"encoding/json"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"

	. "github.com/elastos/Elastos.ELA.Utility/common"
)

func InitMockConfig() {

	mocConfig := []byte("{" +
		"  \"Configuration\": {" +
		"    \"Magic\": 7630402," +
		"    \"Version\": 0," +
		"    \"SeedList\": [" +
		"      \"127.0.0.1:20338\"" +
		"    ]," +
		"    \"NodePort\": 10338," +
		"    \"PrintLevel\": 1," +
		"    \"HttpJsonPort\": 10010," +
		"    \"MainNode\": {" +
		"      \"Rpc\": {" +
		"        \"IpAddress\": \"localhost\"," +
		"        \"HttpJsonPort\": 10038" +
		"      }," +
		"      \"PrintLevel\": 4," +
		"      \"SpvSeedList\": [" +
		"        \"127.0.0.1:20866\"" +
		"      ]" +
		"    }," +
		"    \"SideNodeList\": [" +
		"      {" +
		"        \"Rpc\": {" +
		"          \"IpAddress\": \"localhost\"," +
		"          \"HttpJsonPort\": 20038" +
		"        }," +
		"        \"ExchangeRate\": 1.0," +
		"        \"GenesisBlock\": \"7c1a76281736d40599d6ae347d1bad924ab02b06c6cf9acd84f519dfdeb78d16\"" +
		"      }," +
		"      {" +
		"        \"Rpc\": {" +
		"          \"IpAddress\": \"localhost\"," +
		"          \"HttpJsonPort\": 30038" +
		"        }," +
		"        \"ExchangeRate\": 1.0," +
		"        \"GenesisBlock\": \"7c1a76281736d40599d6ae347d1bad924ab02b06c6cf9acd84f519dfdeb78d33\"" +
		"      }" +
		"    ]," +
		"    \"SyncInterval\": 10000," +
		"    \"SideChainMonitorScanInterval\": 1000" +
		"  }" +
		"}")

	config := ConfigFile{}
	json.Unmarshal(mocConfig, &config)
	Parameters.Configuration = &config.ConfigFile

	for _, node := range Parameters.SideNodeList {
		genesisBytes, err := HexStringToBytes(node.GenesisBlock)
		if err != nil {
			return
		}
		reversedGenesisBytes := BytesReverse(genesisBytes)
		reversedGenesisStr := BytesToHexString(reversedGenesisBytes)
		genesisBlockHash, err := Uint256FromHexString(reversedGenesisStr)
		if err != nil {
			return
		}
		address, err := base.GetGenesisAddress(*genesisBlockHash)
		if err != nil {
			return
		}
		node.GenesisBlockAddress = address
		node.GenesisBlock = reversedGenesisStr
	}
}
