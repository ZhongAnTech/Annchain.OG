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
	og_types "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/common"
	"github.com/tinylib/msgp/msgp"
	"strings"
)

//go:generate msgp

//msgp:tuple Txi
type Txi interface {
	// Implemented by TxBase
	GetType() TxBaseType
	GetHeight() uint64
	GetWeight() uint64
	GetTxHash() og_types.Hash
	GetNonce() uint64
	Parents() []og_types.Hash // Parents returns the common.Hash of txs that it directly proves.
	SetHash(h og_types.Hash)
	String() string
	CalcTxHash() og_types.Hash    // TxHash returns a full tx common.Hash (parents sealed by PoW stage 2)
	CalcMinedHash() og_types.Hash // NonceHash returns the part that needs to be considered in PoW stage 1.
	CalculateWeight(parents Txis) uint64

	SetInValid(b bool)
	InValid() bool

	// implemented by each tx type
	GetBase() *TxBase
	Sender() og_types.Address
	GetSender() og_types.Address
	SetSender(addr og_types.Address)
	Dump() string             // For logger dump
	Compare(tx Txi) bool      // Compare compares two txs, return true if they are the same.
	SignatureTargets() []byte // SignatureTargets only returns the parts that needs to be signed by sender.

	//RawTxi() RawTxi // compressed txi

	// implemented by msgp
	DecodeMsg(dc *msgp.Reader) (err error)
	EncodeMsg(en *msgp.Writer) (err error)
	MarshalMsg(b []byte) (o []byte, err error)
	UnmarshalMsg(bts []byte) (o []byte, err error)
	Msgsize() (s int)
	GetVersion() byte
	ToSmallCaseJson() ([]byte, error)
	IsVerified() verifiedType
	SetVerified(v verifiedType)
}

type RawTxi interface {
	GetType() TxBaseType
	GetHeight() uint64
	GetWeight() uint64
	GetTxHash() og_types.Hash
	GetNonce() uint64
	Parents() common.Hashes // Parents returns the hash of txs that it directly proves.
	SetHash(h og_types.Hash)
	String() string
	CalcTxHash() og_types.Hash    // TxHash returns a full tx hash (parents sealed by PoW stage 2)
	CalcMinedHash() og_types.Hash // NonceHash returns the part that needs to be considered in PoW stage 1.
	CalculateWeight(parents Txis) uint64

	Txi() Txi

	// implemented by msgp
	DecodeMsg(dc *msgp.Reader) (err error)
	EncodeMsg(en *msgp.Writer) (err error)
	MarshalMsg(b []byte) (o []byte, err error)
	UnmarshalMsg(bts []byte) (o []byte, err error)
	Msgsize() (s int)
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

/*func (t Txis) ToTxs() (txs Txs, cps Campaigns, tcs TermChanges, seqs Sequencers) {
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
}*/

//func (t Txis) ToRaw() (txs RawTxs, cps RawCampaigns, tcs RawTermChanges, seqs RawSequencers) {
//	for _, txi := range t {
//		switch tx := txi.(type) {
//		case *Tx:
//			txs = append(txs, tx.RawTx())
//		case *Campaign:
//			cps = append(cps, tx.RawCampaign())
//		case *TermChange:
//			tcs = append(tcs, tx.RawTermChange())
//		case *Sequencer:
//			seqs = append(seqs, tx.RawSequencer())
//		}
//	}
//	if len(txs) == 0 {
//		txs = nil
//	}
//	if len(cps) == 0 {
//		cps = nil
//	}
//	if len(seqs) == 0 {
//		seqs = nil
//	}
//	return
//}
//msgp:tuple RawTxis
//type RawTxis []RawTxi
//
//func (t Txis) RawTxis() RawTxis {
//	var txs RawTxis
//	for _, tx := range t {
//		if tx == nil {
//			continue
//		}
//		txs = append(txs, tx.RawTxi())
//	}
//	if len(txs) == 0 {
//		txs = nil
//	}
//	return txs
//}

//msgp:tuple TxiSmallCaseMarshal
type TxiSmallCaseMarshal struct {
	Txi Txi
}

func (t *TxiSmallCaseMarshal) MarshalJSON() ([]byte, error) {
	if t == nil || t.Txi == nil {
		return nil, nil
	}
	return t.Txi.ToSmallCaseJson()
}
