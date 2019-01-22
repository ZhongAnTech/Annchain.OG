package types

import (
	"fmt"
	"github.com/annchain/OG/common/math"
	"strings"
)

//go:generate msgp

// compress data ,for p2p  , small size
type RawTx struct {
	TxBase
	To    Address
	Value *math.BigInt
}

type RawSequencer struct {
	TxBase
}

type RawSequencers []*RawSequencer

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

func (r RawTxs)Txs() Txs {
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

func (r RawSequencers) Txis() Txis{
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
