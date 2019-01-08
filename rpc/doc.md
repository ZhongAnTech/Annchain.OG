# **RPC API Document** 

## **Get Status**
Get the status of 

**URL**: `/status` 

**Method**: GET

**请求参数**:  

| 参数 | 数据类型 | 是否必填 
| --- | --- | --- 

**请求示例**：
> /status

**返回示例**:
```json
{

}
```
---

## **Get Net Information**
Get information of the network. 

**URL**: `/net_info` 

**Method**: GET

**请求参数**:  

| 参数 | 数据类型 | 是否必填 
| --- | --- | --- 

**请求示例**：
> /net_info

**返回示例**:
```json
{

}
```
---

## **Get Peers Information**
Get information of the peers. 

**URL**: `/peers_info` 

**Method**: GET

**请求参数**:  

| 参数 | 数据类型 | 是否必填 
| --- | --- | --- 

**请求示例**：
> /peers_info

**返回示例**:
```json
{

}
```
---

## **Query Transaction**
Get transaction from og node. 

**URL**: `/transaction` 

**Method**: GET

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
| hash | string | 是 | 必须是可以转成byte数组的 hex string，且不可带有"0x"

**请求示例**：
> /peers_info?hash=69a1379feffe1049e0b45d5dcb131034f79e94cd2ce5085cececb9c4ccdc2be0

**返回示例**:
```json
{

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
| hash | string | 是 | tx的哈希，必须是可以转成byte数组的 hex string，且不可带有"0x"

**请求示例**：
> /confirm?hash=69a1379feffe1049e0b45d5dcb131034f79e94cd2ce5085cececb9c4ccdc2be0

**返回示例**:
```json
{

}
```
---

## **Transactions**
Check if a transaction is been confirmed. 

**Method**: GET

**URL**: 
```
/transactions
```

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
| seq_id | int | 否 | 和 address 两个选一个必填，如果address有值优先获取地址相关的所有交易。
| address | string | 否 | 和 seq_id 两个选一个必填，必须是 hex string.

**请求示例**：
> /confirm?seq_id=123

> /confirm?address=96f4ac2f3215b80ea3a6466ebc1f268f6f1d5406

**返回示例**:
```json
{

}
```
---

## **Genesis**
Check genesis sequencer from OG. 

**Method**: GET

**URL**: 
```
/genesis
```

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---

**请求示例**：
> /genesis

**返回示例**:
```json
{

}
```
---

## **Sequencer**
Check sequencer from OG. 

**Method**: GET

**URL**: 
```
/sequencer
```


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

}
```
---

## **New Transaction**
Send new transaction to OG. 

**Method**: GET / POST

**URL**: 
```
/new_transaction
```

**请求参数**:  

| 参数 | 数据类型 | 是否必填 | 备注
| --- | --- | --- | ---
<!-- todo -->

**请求示例**：
```json
{
    "nonce": "0",
    "from": "0x889e0b36dc6f2c06eb68d9c5f53434e4c42c8d19",
    "to": "0x473c176c84213626588c4d2d7724b9524aaf6f3d",
    "value": "0",
    "signature": "0x421001d20e2dbbd13...",
    "pubkey": "0x0104249f001e59783eb10f1..."
}
```

**返回示例**:
```json
{

}
```
---

