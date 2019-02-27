package types

import (
	"fmt"
	"github.com/annchain/OG/common/math"
	"strings"
)

//go:generate msgp

// compress data ,for p2p  , small size
//msgp:tuple RawTx
type RawTx struct {
	TxBase
	To    Address
	Value *math.BigInt
}

//msgp:tuple RawSequencer
type RawSequencer struct {
	TxBase
}

//msgp:tuple RawCampaign
type RawCampaign struct {
	TxBase
	DkgPublicKey []byte
	Vrf          VrfInfo
}

//msgp:tuple RawTermChange
type RawTermChange struct {
	TxBase
	PkBls  []byte
	SigSet map[Address][]byte
}

//msgp:tuple RawSequencers
type RawSequencers []*RawSequencer

//msgp:tuple RawCampaigns
type RawCampaigns []*RawCampaign

//msgp:tuple RawTermChanges
type RawTermChanges []*RawTermChange

//msgp:tuple RawTxs
type RawTxs []*RawTx

func (t *RawTx) Tx() *Tx {
	if t == nil {
		return nil
	}
	tx := &Tx{
		TxBase: t.TxBase,
		To:     t.To,
		Value:  t.Value,
	}
	tx.From = Signer.AddressFromPubKeyBytes(tx.PublicKey)
	return tx
}

func (t *RawSequencer) Sequencer() *Sequencer {
	if t == nil {
		return nil
	}
	tx := &Sequencer{
		TxBase: t.TxBase,
	}
	tx.Issuer = Signer.AddressFromPubKeyBytes(tx.PublicKey)
	return tx
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

func (r RawTxs) Txs() Txs {
	if len(r) == 0 {
		return nil
	}
	var txs Txs
	for _, v := range r {
		tx := v.Tx()
		txs = append(txs, tx)
	}
	return txs
}

func (r RawTxs) Txis() Txis {
	if len(r) == 0 {
		return nil
	}
	var txis Txis
	for _, v := range r {
		tx := v.Tx()
		txis = append(txis, tx)
	}
	return txis
}

func (r RawSequencers) Sequencers() Sequencers {
	if len(r) == 0 {
		return nil
	}
	var seqs Sequencers
	for _, v := range r {
		seq := v.Sequencer()
		seqs = append(seqs, seq)
	}
	return seqs
}

func (r RawSequencers) Txis() Txis {
	if len(r) == 0 {
		return nil
	}
	var txis Txis
	for _, v := range r {
		seq := v.Sequencer()
		txis = append(txis, seq)
	}
	return txis
}

func (seqs RawSequencers) ToHeaders() SequencerHeaders {
	if len(seqs) == 0 {
		return nil
	}
	var headers SequencerHeaders
	for _, v := range seqs {
		head := NewSequencerHead(v.Hash, v.Height)
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

func (rc *RawCampaign) Campaign() *Campaign {
	if rc == nil {
		return nil
	}
	cp := &Campaign{
		TxBase:       rc.TxBase,
		DkgPublicKey: rc.DkgPublicKey,
		Vrf:          rc.Vrf,
	}
	cp.Issuer = Signer.AddressFromPubKeyBytes(rc.PublicKey)
	return cp
}

func (r *RawTermChange) TermChange() *TermChange {
	if r == nil {
		return nil
	}
	t := &TermChange{
		TxBase: r.TxBase,
		PkBls:  r.PkBls,
		SigSet: r.SigSet,
	}
	t.Issuer = Signer.AddressFromPubKeyBytes(r.PublicKey)
	return t
}

func (r RawCampaigns) Campaigns() Campaigns {
	if len(r) == 0 {
		return nil
	}
	var cs Campaigns
	for _, v := range r {
		c := v.Campaign()
		cs = append(cs, c)
	}
	return cs
}

func (r RawTermChanges) TermChanges() TermChanges {
	if len(r) == 0 {
		return nil
	}
	var cs TermChanges
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

func (r RawTermChanges) Txis() Txis {
	if len(r) == 0 {
		return nil
	}
	var cs Txis
	for _, v := range r {
		c := v.TermChange()
		cs = append(cs, c)
	}
	return cs
}

func (r RawCampaigns) Txis() Txis {
	if len(r) == 0 {
		return nil
	}
	var cs Txis
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

func (r *Txs) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}

func (r *Campaigns) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}

func (r *TermChanges) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}

func (r *Sequencers) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}
