// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tx_types

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/types"
	"golang.org/x/crypto/sha3"
)

//go:generate msgp

//msgp:tuple Archive
type Archive struct {
	types.TxBase
	Data []byte `json:"data"`
}

//msgp:tuple ArchiveJson
type ArchiveJson struct {
	types.TxBaseJson
	Data []byte `json:"data"`
}

func (a *Archive) ToSmallCaseJson() ([]byte, error) {
	if a == nil {
		return nil, nil
	}
	j := ArchiveJson{
		TxBaseJson: *a.TxBase.ToSmallCase(),
		Data:       a.Data,
	}
	return json.Marshal(&j)
}

//msgp:tuple Campaigns
type Archives []*Archive

func (a *Archive) GetBase() *types.TxBase {
	return &a.TxBase
}

func (a *Archive) Sender() common.Address {
	panic("not implemented")
	return common.Address{}
	//return  &Address{}
}

func (tc *Archive) GetSender() *common.Address {
	panic("not implemented")
	return nil
}

func (c *Archive) Compare(tx types.Txi) bool {
	switch tx := tx.(type) {
	case *Campaign:
		if c.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (c *Archive) Dump() string {
	var phashes []string
	for _, p := range c.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	return fmt.Sprintf("hash: %s, pHash: [%s] , nonce: %d  ,Data: %x", c.Hash.Hex(),
		strings.Join(phashes, " ,"), c.AccountNonce, c.Data)
}

func (a *Archive) SignatureTargets() []byte {
	// add parents infornmation.
	panic("not inplemented")
}

func (a *Archive) String() string {
	return fmt.Sprintf("%s-%d-Ac", a.TxBase.String(), a.AccountNonce)
}

func (as Archives) String() string {
	var strs []string
	for _, v := range as {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (a *Archive) RawArchive() *RawArchive {
	if a == nil {
		return nil
	}
	ra := RawArchive{
		Archive: *a,
	}
	return &ra
}

func (cs Archives) RawArchives() RawArchives {
	if len(cs) == 0 {
		return nil
	}
	var rawCps RawArchives
	for _, v := range cs {
		rasSeq := v.RawArchive()
		rawCps = append(rawCps, rasSeq)
	}
	return rawCps
}

func (c *Archive) RawTxi() types.RawTxi {
	return c.RawArchive()
}

func RandomArchive() *Archive {
	return &Archive{TxBase: types.TxBase{
		Hash:        common.RandomHash(),
		Height:      uint64(rand.Int63n(1000)),
		ParentsHash: common.Hashes{common.RandomHash(), common.RandomHash()},
		Type:        types.TxBaseTypeArchive,
		//AccountNonce: uint64(rand.Int63n(50000)),
		Weight: uint64(rand.Int31n(2000)),
	},
		Data: common.RandomHash().ToBytes(),
	}
}

func (t *Archive) CalcTxHash() (hash common.Hash) {
	w := types.NewBinaryWriter()

	for _, ancestor := range t.ParentsHash {
		w.Write(ancestor.Bytes)
	}
	// do not use Height to calculate tx hash.
	w.Write(t.Weight, t.Data, t.CalcMinedHash().Bytes)
	result := sha3.Sum256(w.Bytes())
	hash.MustSetBytes(result[0:], common.PaddingNone)
	return
}

func (t *Archive) SetSender(address common.Address) {
	return
}

// OpStrAndSign 存证字符串跟签名合并
type OpStrAndSign struct {
	OpStr     []byte `json:"op_str"`    /* 存证字符串 */
	Signature string `json:"signature"` /* 签名，十六进制字符串格式 */
}

// NewOpStrAndSign 构造OpStrAndSign
func NewOpStrAndSign(opStr []byte, sign crypto.Signature) *OpStrAndSign {
	return &OpStrAndSign{
		OpStr:     opStr,
		Signature: hexutil.Encode(sign.Bytes),
	}
}

// 存证：type=4
// "data"
//     "type"
//     "transaction"
//     "sequencer"
//     "archive"
//         "type"
//         "hash"
//         "parents"
//         "account_nonce"
//         "mind_nonce"
//         "weight"
// --------"height"
//         "data"
// ++++++++"public_key"
// ++++++++"signature"
// "err"
type ArchiveMsg struct {
	Type         int      `json:"type"`
	Hash         string   `json:"hash"`
	Parents      []string `json:"parents"`
	AccountNonce uint64   `json:"account_nonce"`
	MindNonce    uint64   `json:"mind_nonce"`
	Weight       uint64   `json:"weight"`
	Height       uint64   `json:"height"`
	Data         []byte   `json:"data"`
	PublicKey    string   `json:"public_key"` /* 公钥 */
	Sign         string   `json:"signature"`  /* 签名 */
	OpHash       string   `json:"op_hash"`    /* 存证哈希 */
}

func (t *Archive) ToJsonMsg() ArchiveMsg {
	txMsg := ArchiveMsg{}

	txMsg.Type = int(t.GetType())
	txMsg.Hash = t.GetTxHash().Hex()
	txMsg.AccountNonce = t.GetNonce()
	txMsg.MindNonce = t.MineNonce
	txMsg.Weight = t.GetWeight()
	txMsg.Height = t.Height
	txMsg.Sign = t.Signature.String() /* 填入TxBase的签名 */
	txMsg.Parents = make([]string, 0)
	for _, p := range t.ParentsHash {
		txMsg.Parents = append(txMsg.Parents, p.Hex())
	}
	txMsg.Data = t.Data
	txMsg.PublicKey = string(t.PublicKey) /* 填入TxBase的公钥 */
	txMsg.OpHash = t.GetOpHash().Hex()    /* 填入TxBase的存证哈希 */
	return txMsg
}
