package config

import "encoding/json"

func InitMockConfig() {

	mocConfig := []byte("{" +
		"  \"PrintLevel\": 4," +
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
		"        \"Rate\": 1.0," +
		"        \"GenesisBlockAddress\": \"EMmfgnrDLQmFPBJiWvsyYGV2jzLQY58J4Y\"," +
		"        \"DestroyAddress\": \"EeM7JrxNdi8MzgBfDExcAUTRXgH3vVHn54\"" +
		"      }," +
		"      {" +
		"        \"Rpc\": {" +
		"          \"IpAddress\": \"localhost\"," +
		"          \"HttpJsonPort\": 30038" +
		"        }," +
		"        \"Rate\": 1.0," +
		"        \"GenesisBlockAddress\": \"EMmfgnrDLQmFPBJiWvsyYGV2jzLQY58J4Y\"," +
		"        \"DestroyAddress\": \"EeM7JrxNdi8MzgBfDExcAUTRXgH3vVHn54\"" +
		"      }" +
		"    ]," +
		"    \"SyncInterval\": 10000," +
		"    \"SideChainMonitorScanInterval\": 1000" +
		"  }" +
		"}")

	config := ConfigFile{}
	json.Unmarshal(mocConfig, &config)
	Parameters.Configuration = &config.ConfigFile
}
