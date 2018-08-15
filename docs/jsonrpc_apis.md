Instructions
===============

this is the document of arbiter json rpc interfaces.
it follows json-rpc 2.0 protocol but also keeps compatible with 1.0 version. 
That means both named params and positional params are acceptable.

"id" is optional, which will be sent back in the result samely if you add it in a request. 
It is needed when you want to distinguish different requests.

"jsonrpc" is optional. It tells which version this request uses.
In version 2.0 it is required, while in version 1.0 it does not exist.

#### getinfo  
description: return part of parameters of current arbiter

parameters: none

result: 

| name   | type | description |
| ------ | ---- | ----------- |
| version | int | the version of arbiter | 
| SideChainMonitorScanInterval | int | the interval of side chain monitor scan | 
| ClearTransactionInterval | int | the interval of clear exist cross chain transaction | 
| MinReceivedUsedUtxoMsgNumber | int | the min received used utxo message number | 
| MinOutbound | int | the min connections of neighbor arbiters | 
| MaxConnections | int | the max connections of neighbor arbiters | 
| SideAuxPowFee | int | the side mining fee | 
| MinThreshold | int | the min amount need in side mining account | 
| DepositAmount | int | the amount deposit to side mining account each time | 

arguments sample:
```json
{
  "method":"getinfo"
}
```

result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": {
        "version": 0,
        "SideChainMonitorScanInterval": 1000,
        "ClearTransactionInterval": 60000,
        "MinReceivedUsedUtxoMsgNumber": 2,
        "MinOutbound": 3,
        "MaxConnections": 8,
        "SideAuxPowFee": 50000,
        "MinThreshold": 10000000,
        "DepositAmount": 10000000
    }
}
```
#### getsidemininginfo  
description: return last side mining height

parameters:

| name   | type | description |
| ------ | ---- | ----------- |
| hash | string | the genesis block hash of one side chain| 

result: 

| name   | type | description |
| ------ | ---- | ----------- |
| LastSendSideMiningHeight | int | the height of last send side mining height | 
| LastNotifySideMiningHeight | int | the height of last notify side mining height | 
| LastSubmitAuxpowHeight | int | the height of last submit auxpow height | 

arguments sample:
```json
{
  "method": "getsidemininginfo",
  "params":{
    "hash":"56be936978c261b2e649d58dbfaf3f23d4a868274f5522cd2adb4308a955c4a3"
  }
}
```

result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": {
        "LastSendSideMiningHeight": 6008,
        "LastNotifySideMiningHeight": 6006,
        "LastSubmitAuxpowHeight": 6006
    }
}
```
#### getmainchainblockheight  
description: return current main chain block height of arbiter

parameters: none

arguments sample:
```json
{
  "method": "getmainchainblockheight"
}
```

result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": 6038
}
```
#### getsidechainblockheight  
description: return current side chain block height of arbiter

parameters:

| name   | type | description |
| ------ | ---- | ----------- |
| hash | string | the genesis block hash of one side chain| 

arguments sample:
```json
{
  "method": "getsidechainblockheight",
  "params":{
      "hash":"56be936978c261b2e649d58dbfaf3f23d4a868274f5522cd2adb4308a955c4a3"
    }
}
```

result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": 70
}
```
#### getfinisheddeposittxs  
description: return finished deposit transactions 

parameters:

| name   | type | description |
| ------ | ---- | ----------- |
| succeed | bool | set to get succed or failed deposit transactions | 

result: 

| name   | type | description |
| ------ | ---- | ----------- |
| Transactions | string | the transaction struct of deposit transactions | 
| TransactionHash | string | the deposit transaction from main chain | 
| GenesisAddress | string | the genesis address of side chain | 

arguments sample:
```json
{
  "method": "getfinisheddeposittxs",
  "params":{
    "succeed":"false"
  }
}
```

result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": {
        "Transactions": [
            {
                "Hash": "2aa0dcd14fd517771b14e4f863a6891bf74b22863b44923625f24f04c2b6029e",
                "GenesisBlockAddress": "XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ"
            },
            {
                "Hash": "760908ddc28893163a9de4c4bc5edd8f597c2c9e0607c23bebff489b741e2cb0",
                "GenesisBlockAddress": "XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ"
            },
            {
                "Hash": "efc91df2d8667d260bb2d260a002a50003cc13b79b80ad8fbb327665e0ea36cd",
                "GenesisBlockAddress": "XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ"
            },
            {
                "Hash": "c4e58aa5c9f624f7964ae14d260cb1ff8c227e93d016e35b56fd96cec8d8bcb6",
                "GenesisBlockAddress": "XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ"
            }
        ]
    }
}
```
#### getfinishedwithdrawtxs  
description: return finished withdraw transactions 

parameters:

| name   | type | description |
| ------ | ---- | ----------- |
| succeed | bool | set to get succed or failed withdraw transactions | 

result: 

| name   | type | description |
| ------ | ---- | ----------- |
| Transactions | string | the transaction hashes of withdraw transactions | 

arguments sample:
```json
{
  "method": "getfinishedwithdrawtxs",
  "params":{
    "succeed":"false"
  }
}
```

result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": {
        "Transactions": [
            "2aa0dcd14fd517771b14e4f863a6891bf74b22863b44923625f24f04c2b6029e",
            "760908ddc28893163a9de4c4bc5edd8f597c2c9e0607c23bebff489b741e2cb0"
        ]
    }
}
```
#### getgitversion  
description: return git version of current arbiter

parameters: none

result: 

| name   | type | description |
| ------ | ---- | ----------- |
| version | string | the git version of current arbiter | 

arguments sample:
```json
{
  "method":"getgitversion"
}
```

result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": "ea51-dirty"
}
```
#### getspvheight  
description: return current main chain height of spv

parameters: none

arguments sample:
```json
{
  "method": "getspvheight"
}
```

result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": 2509
}
```
