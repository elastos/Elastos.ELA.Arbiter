package config

import "encoding/json"

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
		"        \"GenesisBlockAddress\": \"XQd1DCi6H62NQdWZQhJCRnrPn7sF9CTjaU\"," +
		"        \"GenesisBlock\": \"168db7dedf19f584cd9acfc6062bb04a92ad1b7d34aed69905d4361728761a7c\"" +
		"      }," +
		"      {" +
		"        \"Rpc\": {" +
		"          \"IpAddress\": \"localhost\"," +
		"          \"HttpJsonPort\": 30038" +
		"        }," +
		"        \"ExchangeRate\": 1.0," +
		"        \"GenesisBlockAddress\": \"XQd1DCi6H62NQdWZQhJCRnrPn7sF9CTjaU\"," +
		"        \"GenesisBlock\": \"168db7dedf19f584cd9acfc6062bb04a92ad1b7d34aed69905d4361728761a7c\"" +
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
