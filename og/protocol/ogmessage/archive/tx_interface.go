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

package archive

import (
	"github.com/annchain/OG/common"
	"github.com/tinylib/msgp/msgp"
)

//go:generate msgp

//msgp:tuple Txi

type RawTxi interface {
	GetType() TxBaseType
	GetHeight() uint64
	GetWeight() uint64
	GetTxHash() common.Hash
	GetNonce() uint64
	Parents() common.Hashes // Parents returns the hash of txs that it directly proves.
	SetHash(h common.Hash)
	String() string
	CalcTxHash() common.Hash    // TxHash returns a full tx hash (parents sealed by PoW stage 2)
	CalcMinedHash() common.Hash // NonceHash returns the part that needs to be considered in PoW stage 1.
	CalculateWeight(parents Txis) uint64

	Txi() Txi

	// implemented by msgp
	DecodeMsg(dc *msgp.Reader) (err error)
	EncodeMsg(en *msgp.Writer) (err error)
	MarshalMsg(b []byte) (o []byte, err error)
	UnmarshalMsg(bts []byte) (o []byte, err error)
	Msgsize() (s int)
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
type RawTxis []RawTxi

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
