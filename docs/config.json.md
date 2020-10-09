# config.json explanation

```json5
{
  "Configuration": {
    "Magic": 2017003,       // Magic Number：Segregation for different subnet. No matter the port number, as long as the magic number not matching, nodes cannot talk to each others
    "Version": 0,
    "NodePort": 20538,      // P2P port number
    "PrintLevel": 1,        // Log level. Level 0 is the highest, 5 is the lowest
    "SpvPrintLevel": 1,     // SPV Log level. Level 0 is the highest, 5 is the lowest
    "HttpJsonPort": 20536,  // RPC port number
    "MainNode": {
      "Rpc": {
        "IpAddress": "127.0.0.1",    // Main ELA Node Ip Address
        "HttpJsonPort": 20336,       // Main ELA Node Rpc port number 
        "User": "USER",              // The username when use rpc interface
        "Pass": "PASS"               // The password when use rpc interface,
      },
      "SpvSeedList": [               // SpvSeedList. spv module use the seed list to discover mainnet peers
        "127.0.0.1:20338",                    
        "node-mainnet-001.elastos.org:20338",
        "node-mainnet-024.elastos.org:20338"
      ],
      "Magic": 2017001,                                             // Main ELA Node Magic Number
      "FoundationAddress": "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta",    // Main ELA Node FoundationAddress
      "DefaultPort": 20338                                          // DefaultPort for spv to connect the main ela node 
    },
    "SideNodeList": [{                
        "Rpc": {                        
          "IpAddress": "127.0.0.1",       // SideChain Node Ip Address
          "HttpJsonPort": 20606,          // SideChain Node Rpc Port 
          "User": "USER",                 // SideChain Node Rpc Username
          "Pass": "PASS"                  // SideChain Node Rpc Password
        },
        "SyncStartHeight": 0,             // The height at which synchronization begins.
        "ExchangeRate": 1.0,              // Sidechain token exchange rate with ELA
        "GenesisBlock": "56be936978c261b2e649d58dbfaf3f23d4a868274f5522cd2adb4308a955c4a3", // SideChain genesis block hash
        "MiningAddr": "EWYdXxK6L8unXcz2Hu2nmLBQLr67Qx5c2b",                                 // Sending sideChain pow transaction address
        "PowChain": true,                                                                   // Indicate if this is a pow sidechain 
        "PayToAddr": "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"                                   // SideChain mining address
      },
      {
        "Rpc": {
          "IpAddress": "127.0.0.1",
          "HttpJsonPort": 20616,
          "User": "USER",
          "Pass": "PASS"
        },
        "SyncStartHeight": 0, 
        "ExchangeRate": 1.0,
        "GenesisBlock": "b569111dfb5e12d40be5cf09e42f7301128e9ac7ab3c6a26f24e77872b9a730e",
        "MiningAddr": "EXeog2edenqtrJM3wnWHmWZzmyataX6pgh",
        "PowChain": true,
        "PayToAddr": "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"
      },
      {
        "Rpc": {
          "IpAddress": "127.0.0.1",
          "HttpJsonPort": 20632,
          "User": "",
          "Pass": ""
        },
        "SyncStartHeight": 0,       // The height at which synchronization begins.
        "ExchangeRate": 1.0,
        "GenesisBlock": "6afc2eb01956dfe192dc4cd065efdf6c3c80448776ca367a7246d279e228ff0a",
        "MiningAddr": "EQ3h7C9hHe1WWDaNyYRtAe3Lx8zV9eHpjM",
        "PowChain": false,
        "PayToAddr": "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"
      }
    ],
    "OriginCrossChainArbiters": [                                                  // The publickey list of arbiters before CRCOnlyDPOSHeight
      "0248df6705a909432be041e0baa25b8f648741018f70d1911f2ed28778db4b8fe4",
      "02771faf0f4d4235744b30972d5f2c470993920846c761e4d08889ecfdc061cddf",
      "0342196610e57d75ba3afa26e030092020aec56822104e465cba1d8f69f8d83c8e",
      "02fa3e0d14e0e93ca41c3c0f008679e417cf2adb6375dd4bbbee9ed8e8db606a56",
      "03ab3ecd1148b018d480224520917c6c3663a3631f198e3b25cf4c9c76786b7850"
    ],
    "CRCCrossChainArbiters": [                                                     // The crc arbiters after CRCOnlyDPOSHeight 
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
      "02a0aa9eac0e168f3474c2a0d04e50130833905740a5270e8a44d6c6e85cf6d98c"
    ],
    "DPoSNetAddress": "127.0.0.1:20339",            // The address used for arbiter to connect with the main ela node
    "CRCOnlyDPOSHeight": 343400,                    // The height start DPOS by CRC producers
    "MinThreshold": 1000000,                        // The minimum value for warning the mining address don't have enough coin
    "DepositAmount": 1000000,                       // The Amount of money to deposit when minthreshold reaches
    "SyncInterval": 1000,                           // Arbiter syncing with mainchain interval
    "SideChainMonitorScanInterval": 1000,           // Arbiter syncing with sidechain interval
    "ClearTransactionInterval": 60000,              // Clear handled transaction interval 
    "MinOutbound": 3,
    "MaxConnections": 8,
    "SideAuxPowFee": 50000,                         // Sidechain pow transaction fee
    "MaxTxsPerWithdrawTx": 1000,                    // Sidechain withdraw transaction process limit per block
    "RpcConfiguration": {                           // Arbiter RPC Configuration 
      "User": "USER",
      "Pass": "PASS",
      "WhiteIPList": [
        "IP"
      ]
    }
  }
}

```