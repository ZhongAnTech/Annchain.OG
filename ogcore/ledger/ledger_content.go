package core

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
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
