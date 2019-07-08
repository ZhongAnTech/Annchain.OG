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
package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/common/hexutil"
	"strings"

	"github.com/tinylib/msgp/msgp"
	"golang.org/x/crypto/sha3"
)

//go:generate msgp
type TxBaseType uint16

const (
	TxBaseTypeNormal TxBaseType = iota
	TxBaseTypeSequencer
	TxBaseTypeCampaign
	TxBaseTypeTermChange
	TxBaseTypeArchive
	TxBaseAction
)

func (t TxBaseType) String() string {
	switch t {
	case TxBaseTypeNormal:
		return "TX"
	case TxBaseTypeSequencer:
		return "SQ"
	case TxBaseTypeCampaign:
		return "CP"
	case TxBaseTypeTermChange:
		return "TC"
	case TxBaseTypeArchive:
		return "AC"
	case TxBaseAction:
		return "ATX"
	default:
		return "NA"
	}
}

// Here indicates what fields should be concerned during hash calculation and signature generation
// |      |                   | Signature target |     NonceHash(Slow) |    TxHash(Fast) |
// |------|-------------------|------------------|---------------------|-----------------|
// | Base | ParentsHash       |                  |                     |               1 |
// | Base | Height            |                  |                     |                 |
// | Base | PublicKey         |                  |                   1 | 1 (nonce hash)  |
// | Base | Signature         |                  |                   1 | 1 (nonce hash)  |
// | Base | MinedNonce        |                  |                   1 | 1 (nonce hash)  |
// | Base | AccountNonce      |                1 |                     |                 |
// | Tx   | From              |                1 |                     |                 |
// | Tx   | To                |                1 |                     |                 |
// | Tx   | Value             |                1 |                     |                 |
// | Tx   | Data              |                1 |                     |                 |
// | Seq  | Id                |                1 |                     |                 |
// | Seq  | ContractHashOrder |                1 |                     |                 |

//msgp:tuple Txi
type Txi interface {
	// Implemented by TxBase
	GetType() TxBaseType
	GetHeight() uint64
	GetWeight() uint64
	GetTxHash() Hash
	GetNonce() uint64
	Parents() Hashes // Parents returns the hash of txs that it directly proves.
	SetHash(h Hash)
	String() string
	CalcTxHash() Hash    // TxHash returns a full tx hash (parents sealed by PoW stage 2)
	CalcMinedHash() Hash // NonceHash returns the part that needs to be considered in PoW stage 1.
	CalculateWeight(parents Txis) uint64

	SetInValid(b bool)
	InValid() bool

	// implemented by each tx type
	GetBase() *TxBase
	Sender() Address
	GetSender() *Address
	SetSender(addr Address)
	Dump() string             // For logger dump
	Compare(tx Txi) bool      // Compare compares two txs, return true if they are the same.
	SignatureTargets() []byte // SignatureTargets only returns the parts that needs to be signed by sender.

	RawTxi() RawTxi // compressed txi

	// implemented by msgp
	DecodeMsg(dc *msgp.Reader) (err error)
	EncodeMsg(en *msgp.Writer) (err error)
	MarshalMsg(b []byte) (o []byte, err error)
	UnmarshalMsg(bts []byte) (o []byte, err error)
	Msgsize() (s int)
	GetVersion() byte

	ToSmallCaseJson() ([]byte, error)
}

//msgp:tuple TxBase
type TxBase struct {
	Type         TxBaseType
	Hash         Hash
	ParentsHash  Hashes
	AccountNonce uint64
	Height       uint64
	PublicKey    PublicKey //
	Signature    hexutil.Bytes
	MineNonce    uint64
	Weight       uint64
	inValid      bool
	Version      byte
}

//msgp:tuple TxBaseJson
type TxBaseJson struct {
	Type         TxBaseType    `json:"type"`
	Hash         Hash          `json:"hash"`
	ParentsHash  Hashes        `json:"parents_hash"`
	AccountNonce uint64        `json:"account_nonce"`
	Height       uint64        `json:"height"`
	PublicKey    PublicKey     `json:"public_key"`
	Signature    hexutil.Bytes `json:"signature"`
	MineNonce    uint64        `json:"mine_nonce"`
	Weight       uint64        `json:"weight"`
	inValid      bool          `json:"in_valid"`
	Version      byte          `json:"version"`
}

type TxiSmallCaseMarshal struct {
	Txi Txi
}

func (t *TxBase) GetVersion() byte {
	return t.Version
}

func (t *TxiSmallCaseMarshal) MarshalJSON() ([]byte, error) {
	if t == nil || t.Txi == nil {
		return nil, nil
	}
	return t.Txi.ToSmallCaseJson()
}

func (t *TxBase) ToSmallCase() *TxBaseJson {
	if t == nil {
		return nil
	}
	b := TxBaseJson{
		Type:         t.Type,
		Height:       t.Height,
		Hash:         t.Hash,
		ParentsHash:  t.ParentsHash,
		AccountNonce: t.AccountNonce,
		PublicKey:    t.PublicKey,
		Signature:    t.Signature,
		MineNonce:    t.MineNonce,
		Weight:       t.Weight,
		inValid:      t.inValid,
	}
	return &b
}

func (t *TxBase) ToSmallCaseJson() ([]byte, error) {
	b := t.ToSmallCase()
	return json.Marshal(b)
}

func (t *TxBase) SetInValid(b bool) {
	t.inValid = b
}

func (t *TxBase) InValid() bool {
	return t.inValid
}

func (t *TxBase) GetType() TxBaseType {
	return t.Type
}

func (t *TxBase) GetHeight() uint64 {
	return t.Height
}

func (t *TxBase) GetWeight() uint64 {
	return t.Weight
}

func (t *TxBase) GetTxHash() Hash {
	return t.Hash
}

func (t *TxBase) GetNonce() uint64 {
	return t.AccountNonce
}

func (t *TxBase) Parents() Hashes {
	return t.ParentsHash
}

func (t *TxBase) SetHash(hash Hash) {
	t.Hash = hash
}

func (t *TxBase) String() string {
	return fmt.Sprintf("%d-[%.10s]-%dw", t.Height, t.GetTxHash().Hex(), t.Weight)
}

func (t *TxBase) CalcTxHash() (hash Hash) {
	var buf bytes.Buffer

	for _, ancestor := range t.ParentsHash {
		panicIfError(binary.Write(&buf, binary.BigEndian, ancestor.Bytes))
	}
	// do not use Height to calculate tx hash.
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Weight))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.CalcMinedHash().Bytes))

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:], PaddingNone)
	return
}

func (t *TxBase) CalcMinedHash() (hash Hash) {
	var buf bytes.Buffer
	if !Signer.CanRecoverPubFromSig() {
		panicIfError(binary.Write(&buf, binary.BigEndian, t.PublicKey))
	}
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Signature))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.MineNonce))

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:], PaddingNone)
	return
}

//CalculateWeight  a core algorithm for tx sorting,
//a tx's weight must bigger than any of it's parent's weight  and bigger than any of it's elder transaction's
func (t *TxBase) CalculateWeight(parents Txis) uint64 {
	var maxWeight uint64
	for _, p := range parents {
		if p.GetWeight() > maxWeight {
			maxWeight = p.GetWeight()
		}
	}
	return maxWeight + 1
}

type Txis []Txi

func (t Txis) String() string {
	var strs []string
	for _, v := range t {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (t Txis) Len() int {
	return len(t)
}

func (t Txis) Less(i, j int) bool {
	if t[i].GetWeight() < t[j].GetWeight() {
		return true
	}
	if t[i].GetWeight() > t[j].GetWeight() {
		return false
	}
	if t[i].GetNonce() < t[j].GetNonce() {
		return true
	}
	if t[i].GetNonce() > t[j].GetNonce() {
		return false
	}
	if t[i].GetTxHash().Cmp(t[j].GetTxHash()) < 0 {
		return true
	}
	return false
}

func (t Txis) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t Txis) ToTxs() (txs Txs, cps Campaigns, tcs TermChanges, seqs Sequencers) {
	for _, txi := range t {
		switch tx := txi.(type) {
		case *Tx:
			txs = append(txs, tx)
		case *Campaign:
			cps = append(cps, tx)
		case *TermChange:
			tcs = append(tcs, tx)
		case *Sequencer:
			seqs = append(seqs, tx)
		}
	}
	if len(txs) == 0 {
		txs = nil
	}
	if len(cps) == 0 {
		cps = nil
	}
	if len(seqs) == 0 {
		seqs = nil
	}
	return
}

func (t Txis) ToRaw() (txs RawTxs, cps RawCampaigns, tcs RawTermChanges, seqs RawSequencers) {
	for _, txi := range t {
		switch tx := txi.(type) {
		case *Tx:
			txs = append(txs, tx.RawTx())
		case *Campaign:
			cps = append(cps, tx.RawCampaign())
		case *TermChange:
			tcs = append(tcs, tx.RawTermChange())
		case *Sequencer:
			seqs = append(seqs, tx.RawSequencer())
		}
	}
	if len(txs) == 0 {
		txs = nil
	}
	if len(cps) == 0 {
		cps = nil
	}
	if len(seqs) == 0 {
		seqs = nil
	}
	return
}

func (t Txis) RawTxis() RawTxis {
	var txs RawTxis
	for _, tx := range t {
		if tx == nil {
			continue
		}
		txs = append(txs, tx.RawTxi())
	}
	if len(txs) == 0 {
		txs = nil
	}
	return txs
}

func (t Txis) TxisMarshaler() TxisMarshaler {
	var txs TxisMarshaler
	for _, tx := range t {
		txs.Append(tx)
	}
	if len(txs) == 0 {
		txs = nil
	}
	return txs
}
