package core

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/types"
)

//go:generate msgp

//msgp:tuple LedgerContentTx
type LedgerContentTx struct {
	Hash         common.Hash
	ParentsHash  []common.Hash
	MineNonce    uint64
	AccountNonce uint64
	From         common.Address
	To           common.Address
	Value        *math.BigInt
	TokenId      int32
	PublicKey    []byte
	Data         []byte
	Signature    []byte
	Height       uint64
	Weight       uint64
}

func (z *LedgerContentTx) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *LedgerContentTx) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func NewTxFromLedgerContentTx(tx *LedgerContentTx) types.Tx {
	return types.Tx{
		Hash:         tx.Hash,
		ParentsHash:  tx.ParentsHash,
		MineNonce:    tx.MineNonce,
		AccountNonce: tx.AccountNonce,
		From:         tx.From,
		To:           tx.To,
		Value:        tx.Value,
		TokenId:      tx.TokenId,
		Data:         tx.Data,
		PublicKey:    crypto.PublicKeyFromRawBytes(tx.PublicKey),
		Signature:    crypto.SignatureFromRawBytes(tx.PublicKey),
		Height:       tx.Height,
		Weight:       tx.Weight,
	}
}

func NewLedgerContentTxFromTx(tx *types.Tx) LedgerContentTx {
	return LedgerContentTx{
		Hash:         tx.Hash,
		ParentsHash:  tx.ParentsHash,
		MineNonce:    tx.MineNonce,
		AccountNonce: tx.AccountNonce,
		From:         tx.From,
		To:           tx.To,
		Value:        tx.Value,
		TokenId:      tx.TokenId,
		PublicKey:    tx.PublicKey.ToBytes(),
		Data:         tx.Data,
		Signature:    tx.Signature.ToBytes(),
		Height:       tx.Height,
		Weight:       tx.Weight,
	}
}

//msgp:tuple LedgerContentSequencer
type LedgerContentSequencer struct {
	Hash         common.Hash
	ParentsHash  []common.Hash
	MineNonce    uint64
	AccountNonce uint64
	Issuer       common.Address
	PublicKey    []byte
	Signature    []byte
	StateRoot    common.Hash
	Height       uint64
	Weight       uint64
}

func (z *LedgerContentSequencer) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *LedgerContentSequencer) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func NewLedgerContentSequencerFromSequencer(seq *types.Sequencer) LedgerContentSequencer {
	return LedgerContentSequencer{
		Hash:         seq.Hash,
		ParentsHash:  seq.ParentsHash,
		MineNonce:    seq.MineNonce,
		AccountNonce: seq.AccountNonce,
		Issuer:       seq.Issuer,
		PublicKey:    seq.PublicKey,
		Signature:    seq.Signature,
		StateRoot:    seq.StateRoot,
		Height:       seq.Height,
		Weight:       seq.Weight,
	}
}

func NewSequencerFromLedgerContentSequencer(seq *LedgerContentSequencer) types.Sequencer {
	return types.Sequencer{
		Hash:         seq.Hash,
		ParentsHash:  seq.ParentsHash,
		MineNonce:    seq.MineNonce,
		AccountNonce: seq.AccountNonce,
		Issuer:       seq.Issuer,
		PublicKey:    seq.PublicKey,
		Signature:    seq.Signature,
		StateRoot:    seq.StateRoot,
		Height:       seq.Height,
		Weight:       seq.Weight,
	}
}