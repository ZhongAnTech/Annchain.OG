# **RPC API Document** 

## **Get Status**
Get the status of 

**URL**:
```
/status
```

**Method**: GET

**请求参数**:  
无

**请求示例**：
> /status

**返回示例**:
```json
{
    "data":{
        "node_info":{
            "id":"99fa33...",
            "short_id":"99fa3376342834b0",
            "name":"og",
            "enode":"enode://99fa3376342...2b8d6@0.0.0.0:8001",
            "ip":"0.0.0.0",
            "ports":{"discovery":8001,"listener":8001},
            "listenAddr":"[::]:8001",
            "protocols":{
                "og":{
                    "network":1,
                    "difficulty":null,
                    "genesis":"0x1dfb...d3736e",
                    "head":"0x38ed1158...e7e2"
                }
            }
        },
        "peers_info":[]
    },
    "message":""
}
```
---

## **Get Net Information**
Get information of the network. 

**URL**:
```
/net_info
```

**Method**: GET

**请求参数**:  
无

**请求示例**：
> /net_info

**返回示例**:
```json
{
    "data":{
        "id":"99fa3...4f2b8d6",
        "short_id":"99fa3376342834b0",
        "name":"og",
        "enode":"enode://99fa337634...2b8d6@0.0.0.0:8001",
        "ip":"0.0.0.0",
        "ports":{"discovery":8001,"listener":8001},
        "listenAddr":"[::]:8001",
        "protocols":{
            "og":{
                "network":1,
                "difficulty":null,
                "genesis":"0x1dfb...36e",
                "head":"0x3e...15"
            }
        }
    },
    "message":""
}
```
---

## **Get Peers Information**
Get information of the peers. 

**URL**:
```
/peers_info
```

**Method**: GET

**请求参数**:  
无

**请求示例**：
> /peers_info

**返回示例**:
```json
{
    "data":[],
    "message":""
}
```
---

## **Query Transaction**
Get transaction from og node. 

**URL**: 
```
/transaction
```

**Method**: GET

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
| hash | string | 是 | 必须是可以转成byte数组的 hex string

**请求示例**：
> /transaction?hash=69a1379feffe1049e0b45d5dcb131034f79e94cd2ce5085cececb9c4ccdc2be0

**返回示例**:
```json
{
    "data":{
        "Type":0,
        "Hash":"0x22359bb1c...8c56",
        "ParentsHash":["0xce63a703e0a...b9990509"],
        "AccountNonce":10,
        "Height":1,
        "PublicKey":"BKVH401d4...eGR+I=",
        "Signature":"HzgliZb...YpzdgE=",
        "MineNonce":1,
        "From":"0x96f4ac2f3215b80ea3a6466ebc1f268f6f1d5406",
        "To":"0xa70c8a9485441f6fa2141d550c26d793107d3dea",
        "Value":"0",
        "Data":null
    },
    "message":""
}
```
---

## **Check Confirm**
Check if a transaction is been confirmed. 

**URL**: 
```
/confirm
```

**Method**: GET

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
| hash | string | 是 | tx的哈希，必须是可以转成byte数组的 hex string

**请求示例**：
> /confirm?hash=69a1379feffe1049e0b45d5dcb131034f79e94cd2ce5085cececb9c4ccdc2be0

**返回示例**:
```json
{
    "data":true,
    "message":""
}
```
---

## **Transactions**
Check if a transaction is been confirmed. 

**URL**: 
```
/transactions
```

**Method**: GET

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
| seq_id | int string | 否 | 和 address 两个选一个必填，如果address有值优先获取地址相关的所有交易。
| address | string | 否 | 和 seq_id 两个选一个必填，必须是 hex string.

**请求示例**：
> /transactions?seq_id=123

> /transactions?address=96f4ac2f3215b80ea3a6466ebc1f268f6f1d5406

**返回示例**:
```json
{
	"data":{
		"total":12,
		"txs":[
			{...},
			{...},
			{...},
			{...},
			{...},
			{...},
			{...},
			{...},
			{...},
			{...},
			{...},
			{...}
		]},
	"message":""
}
```
---

## **Genesis**
Check genesis sequencer from OG. 

**URL**: 
```
/genesis
```

**Method**: GET

**请求参数**:
无

**请求示例**：
> /genesis

**返回示例**:
```json
{
    "data":{
        "Type":1,
        "Hash":"0x1dfb6fea83e3d62af98d72255527a677dbaf3ba4f98c80ae0ea9e3db97d3736e",
        "ParentsHash":null,
        "AccountNonce":0,
        "Height":0,
        "PublicKey":"s+G4MG4bqxXtUaTCSwhlUGd7qZzWKDWWUxajZBno9ZzmojKJIYLadAGjKQZuj+KvYHKHE55jfTFL8NYcudHH7g==",
        "Signature":"MEQCIBIwK9fJUfy/7yZG2Zb6QnCaPMNd/K9ID6Tw+HgmRVhdAiBCTXEC2on0R7KMU6rjiKzwulcAjIBI9eNNwRdlscq39g==",
        "MineNonce":0,
        "Id":0,"
        Issuer":"0x0000000000000000000000000000000000000000",
        "ContractHashOrder":[]
    },
    "message":""
}
```
---

## **Sequencer**
Check sequencer from OG. 

**URL**: 
```
/sequencer
```

**Method**: GET

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
| hash | string | 否 | 必须是hex string，和seq_id字段二选一，都存在的话优先 seq_id，两字段都不存在则返回最新的sequencer
| seq_id | int | 否 | 和 hash 字段二选一，两字段都不存在则返回最新的sequencer

**请求示例**：
> /sequencer?hash=69a1379feffe1049e0b45d5dcb131034f79e94cd2ce5085cececb9c4ccdc2be0

> /sequencer?seq_id=123

**返回示例**:
```json
{
    "data":{
        "Type":1,
        "Hash":"0x5bcb676788cd6...6cf5",
        "ParentsHash":[
            "0xfb9aa6509b5...61b8c233b4d",
            "0x363e56d1a0f...fdaef3b1e46"
        ],
        "AccountNonce":225,
        "Height":12,
        "PublicKey":"BIDG6ARHwZ...0gHt8RtnRHzrI=","Signature":"AcRW9jpW...o9dQE=",
        "MineNonce":1,
        "Id":12,
        "Issuer":"0x7349f7a6f622378d5fb0e2c16b9d4a3e5237c187",
        "ContractHashOrder":null
    },
    "message":""
}
```
---

## **New Transaction**
Send new transaction to OG. 

**URL**: 
```
/new_transaction
```

**Method**: GET / POST

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
| nonce | int string | 是 |
| from | hex string | 是 |
| to | hex string | 否 | 创建合约时可以置空
| value | int string | 是 | 不转账时填0
| signature | hex string | 是 |
| pubkey | hex string | 是 |
| data | hex string | 否 | 

**请求示例**：
```json
{
    "nonce": "0",
    "from": "0x889e0b36dc6f2c06eb68d9c5f53434e4c42c8d19",
    "to": "0x473c176c84213626588c4d2d7724b9524aaf6f3d",
    "value": "0",
    "signature": "0x421001d20e2dbbd13...",
    "pubkey": "0x0104249f001e59783eb10f1...",
    "data": "0x5682aec..."
}
```

**返回示例**:
```json
{
    "data":"0xb4d525888e28119419f8ad1ccb837d899c17c1680f3bb4cb184471313439f570",
    "message":""
}
```
---

## **New Account**
Generage a random key pair. 

**URL**: 
```
/new_account
```

**Method**: POST

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
| algorithm | string | 是 | 签名类型（ed25519, secp256k1）

**请求示例**：
```json
{
    "algorithm": "secp256k1",
}
```

**返回示例**:
```json
{

}
```
---

## **Auto Tx**
TODO 

**URL**: 
```
/auto_tx
```

**Method**: GET

---

## **Query Nonce**
Get latest nonce of a specific address. 

**URL**: 
```
/query_nonce
```

**Method**: GET

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
| address | hex string | 是 | 

**请求示例**：
> /query_nonce?address=96f4ac2f3215b80ea3a6466ebc1f268f6f1d5406

**返回示例**:
```json
{
    "data":144,
    "message":""
}
```
---

## **Query Balance**
Get current balance of a specific address. 

**URL**: 
```
/query_balance
```

**Method**: GET

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
| address | hex string | 是 | 

**请求示例**：
> /query_balance?address=96f4ac2f3215b80ea3a6466ebc1f268f6f1d5406

**返回示例**:
```json
{
    "data":{
        "balance":"8888888",
        "address":"0x96f4ac2f3215b80ea3a6466ebc1f268f6f1d5406"
    },
    "message":""
}
```
---

## **Query Receipt**
Get receipt of a transaction. 

**URL**: 
```
/query_receipt
```

**Method**: GET

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
| hash | hex string | 是 | 

**请求示例**：
> /query_receipt?hash=0x0a0e69f4bd4c027e8ec0d6ab20eda7c8558c9a5ea690aa25b5e1cd72c67f444a

**返回示例**:
```json
{
    "data":{
        "tx_hash":"0x0a0e69...67f444a",
        "status":1,
        "result":"",
        "contract_address":"0x0000...0000000"
    },
    "message":""
}
```
---



