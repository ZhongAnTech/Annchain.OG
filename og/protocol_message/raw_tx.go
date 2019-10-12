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
package protocol_message

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/vrf"
	"github.com/annchain/OG/types/tx_types"

	//"github.com/annchain/OG/consensus/vrf"
	"github.com/annchain/OG/types"
	"strings"

	"github.com/annchain/OG/common/math"
)

//go:generate msgp

// compress data ,for p2p  , small size
//msgp:tuple RawTx
type RawTx struct {
	types.TxBase
	To      common.Address
	Value   *math.BigInt
	Data    []byte
	TokenId int32
}

//msgp:tuple RawActionTx
type RawActionTx struct {
	types.TxBase
	Action     uint8
	ActionData tx_types.ActionData
}

//msgp:tuple RawSequencer
type RawSequencer struct {
	types.TxBase
	BlsJointSig    []byte
	BlsJointPubKey []byte
	StateRoot      common.Hash
}

//msgp:tuple RawCampaign
type RawCampaign struct {
	types.TxBase
	DkgPublicKey []byte
	Vrf          vrf.VrfInfo
}

//msgp:tuple RawTermChange
type RawTermChange struct {
	types.TxBase
	TermId uint64
	PkBls  []byte
	SigSet []*tx_types.SigSet
}

//msgp:tuple RawArchive
type RawArchive struct {
	tx_types.Archive
}

//msgp:tuple RawArchive
type RawArchives []*RawArchive

//msgp:tuple RawSequencers
type RawSequencers []*RawSequencer

//msgp:tuple RawCampaigns
type RawCampaigns []*RawCampaign

//msgp:tuple RawTermChanges
type RawTermChanges []*RawTermChange

//msgp:tuple RawActionTxs
type RawActionTxs []*RawActionTx

//msgp:tuple RawTxs
type RawTxs []*RawTx

func (t *RawTx) Tx() *tx_types.Tx {
	if t == nil {
		return nil
	}
	tx := &tx_types.Tx{
		TxBase:  t.TxBase,
		To:      t.To,
		Value:   t.Value,
		Data:    t.Data,
		TokenId: t.TokenId,
	}
	if !types.CanRecoverPubFromSig {
		tx.SetSender(crypto.Signer.AddressFromPubKeyBytes(tx.PublicKey))
	}
	return tx
}

func (t *RawSequencer) Sequencer() *tx_types.Sequencer {
	if t == nil {
		return nil
	}
	tx := &tx_types.Sequencer{
		TxBase:         t.TxBase,
		BlsJointPubKey: t.BlsJointPubKey,
		BlsJointSig:    t.BlsJointSig,
		StateRoot:      t.StateRoot,
	}
	if !types.CanRecoverPubFromSig {
		addr := crypto.Signer.AddressFromPubKeyBytes(tx.PublicKey)
		tx.Issuer = &addr
	}
	return tx
}

func (t *RawActionTx) ActionTx() *tx_types.ActionTx {
	if t == nil {
		return nil
	}
	if t == nil {
		return nil
	}
	tx := &tx_types.ActionTx{
		TxBase:     t.TxBase,
		Action:     t.Action,
		ActionData: t.ActionData,
	}

	if !types.CanRecoverPubFromSig {
		addr := crypto.Signer.AddressFromPubKeyBytes(tx.PublicKey)
		tx.From = &addr
	}
	return tx
}

func (t *RawActionTx) String() string {
	return fmt.Sprintf("%s-%d-rawATX", t.TxBase.String(), t.AccountNonce)
}

func (t *RawTx) String() string {
	return fmt.Sprintf("%s-%d-RawTx", t.TxBase.String(), t.AccountNonce)
}

func (t *RawSequencer) String() string {
	return fmt.Sprintf("%s-%d_%d-RawSeq", t.TxBase.String(), t.AccountNonce, t.Height)
}

func (t *RawTermChange) String() string {
	return fmt.Sprintf("%s-%d_%d-RawTC", t.TxBase.String(), t.AccountNonce, t.Height)
}

func (t *RawCampaign) String() string {
	return fmt.Sprintf("%s-%d_%d-RawCP", t.TxBase.String(), t.AccountNonce, t.Height)
}

func (r RawTxs) Txs() tx_types.Txs {
	if len(r) == 0 {
		return nil
	}
	var txs tx_types.Txs
	for _, v := range r {
		tx := v.Tx()
		txs = append(txs, tx)
	}
	return txs
}

func (r RawTxs) Txis() types.Txis {
	if len(r) == 0 {
		return nil
	}
	var txis types.Txis
	for _, v := range r {
		tx := v.Tx()
		txis = append(txis, tx)
	}
	return txis
}

func (r RawSequencers) Sequencers() tx_types.Sequencers {
	if len(r) == 0 {
		return nil
	}
	var seqs tx_types.Sequencers
	for _, v := range r {
		seq := v.Sequencer()
		seqs = append(seqs, seq)
	}
	return seqs
}

func (r RawSequencers) Txis() types.Txis {
	if len(r) == 0 {
		return nil
	}
	var txis types.Txis
	for _, v := range r {
		seq := v.Sequencer()
		txis = append(txis, seq)
	}
	return txis
}

func (seqs RawSequencers) ToHeaders() tx_types.SequencerHeaders {
	if len(seqs) == 0 {
		return nil
	}
	var headers tx_types.SequencerHeaders
	for _, v := range seqs {
		head := tx_types.NewSequencerHead(v.Hash, v.Height)
		headers = append(headers, head)
	}
	return headers
}

func (r RawTxs) String() string {
	var strs []string
	for _, v := range r {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (r RawSequencers) String() string {
	var strs []string
	for _, v := range r {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (rc *RawCampaign) Campaign() *tx_types.Campaign {
	if rc == nil {
		return nil
	}
	cp := &tx_types.Campaign{
		TxBase:       rc.TxBase,
		DkgPublicKey: rc.DkgPublicKey,
		Vrf:          rc.Vrf,
	}
	if !types.CanRecoverPubFromSig {
		addr := crypto.Signer.AddressFromPubKeyBytes(rc.PublicKey)
		cp.Issuer = &addr
	}
	return cp
}

func (r *RawTermChange) TermChange() *tx_types.TermChange {
	if r == nil {
		return nil
	}
	t := &tx_types.TermChange{
		TxBase: r.TxBase,
		PkBls:  r.PkBls,
		SigSet: r.SigSet,
		TermID: r.TermId,
	}
	if !types.CanRecoverPubFromSig {
		addr := crypto.Signer.AddressFromPubKeyBytes(r.PublicKey)
		t.Issuer = &addr
	}
	return t
}

func (r RawCampaigns) Campaigns() tx_types.Campaigns {
	if len(r) == 0 {
		return nil
	}
	var cs tx_types.Campaigns
	for _, v := range r {
		c := v.Campaign()
		cs = append(cs, c)
	}
	return cs
}

func (r RawTermChanges) TermChanges() tx_types.TermChanges {
	if len(r) == 0 {
		return nil
	}
	var cs tx_types.TermChanges
	for _, v := range r {
		c := v.TermChange()
		cs = append(cs, c)
	}
	return cs
}

func (r RawTermChanges) String() string {
	var strs []string
	for _, v := range r {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (r RawCampaigns) String() string {
	var strs []string
	for _, v := range r {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (r RawTermChanges) Txis() types.Txis {
	if len(r) == 0 {
		return nil
	}
	var cs types.Txis
	for _, v := range r {
		c := v.TermChange()
		cs = append(cs, c)
	}
	return cs
}

func (r RawCampaigns) Txis() types.Txis {
	if len(r) == 0 {
		return nil
	}
	var cs types.Txis
	for _, v := range r {
		c := v.Campaign()
		cs = append(cs, c)
	}
	return cs
}

func (r *RawTxs) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}

func (r *RawSequencers) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}

func (r *RawCampaigns) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}

func (r *RawTermChanges) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}

type TxisMarshaler []*tx_types.RawTxMarshaler

func (t *TxisMarshaler) Append(tx types.Txi) {
	if tx == nil {
		return
	}
	raw := tx.RawTxi()
	if raw == nil {
		return
	}
	m := tx_types.RawTxMarshaler{raw}
	if t == nil {
		panic("t is nil ")
	}
	*t = append(*t, &m)
}

func (t TxisMarshaler) Len() int {
	if t == nil {
		return 0
	}
	return len(t)
}

func (t TxisMarshaler) String() string {
	var strs []string
	for _, v := range t {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (t TxisMarshaler) Txis() types.Txis {
	if t == nil {
		return nil
	}
	var txis types.Txis
	for _, v := range t {
		if v == nil {
			continue
		}
		txis = append(txis, v.Txi())
	}
	return txis
}

func (t *RawTx) Txi() types.Txi {
	return t.Tx()
}

func (t *RawSequencer) Txi() types.Txi {
	return t.Sequencer()
}

func (t *RawTermChange) Txi() types.Txi {
	return t.TermChange()
}

func (t *RawCampaign) Txi() types.Txi {
	return t.Campaign()
}

func (a *RawArchive) Txi() types.Txi {
	return &a.Archive
}

func (a *RawActionTx) Txi() types.Txi {
	return a.ActionTx()
}

//func (t *RawTx) Dump() string  {
//	return t.Tx().Dump()
//}
//
//func (t *RawSequencer) Dump() string {
//	return t.Sequencer().Dump()
//}
//
//func (t *RawTermChange) Dump() string {
//	return t.TermChange().Dump()
//}
//
//func (t *RawCampaign) Dump() string {
//	return t.Campaign().Dump()
//}
//
//func (t*RawCampaign)GetBase() *TxBase{
//	return t.Campaign().GetBase()
//}
//func (t*RawTermChange)GetBase() *TxBase{
//	return t.TermChange().GetBase()
//
//}
//func (t*RawTx)GetBase() *TxBase{
//	return t.Tx().GetBase()
//
//}
//func (t*RawSequencer)GetBase() *TxBase{
//  return t.Sequencer().GetBase()
//}
//
//func (t*RawCampaign)Sender() common.Address{
//	return t.Campaign().Sender()
//}
//func (t*RawTermChange)Sender() common.Address{
//	return t.TermChange().Sender()
//
//}
//func (t*RawTx)Sender() common.Address{
//	return t.Tx().Sender()
//
//}
//func (t*RawSequencer)Sender() common.Address{
//	return t.Sequencer().Sender()
//}
//
//func (t*RawCampaign)SignatureTargets() []byte{
//	return t.Campaign().SignatureTargets()
//}
//func (t*RawTermChange)SignatureTargets() []byte{
//	return t.TermChange().SignatureTargets()
//
//}
//func (t*RawTx)SignatureTargets() []byte{
//	return t.Tx().SignatureTargets()
//
//}
//func (t*RawSequencer)SignatureTargets() []byte{
//	return t.Sequencer().SignatureTargets()
//}
