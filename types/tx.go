package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strings"

	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
)

//go:generate msgp

//msgp:tuple Txs
type Txs []*Tx

//msgp:tuple Tx
type Tx struct {
	TxBase
	From  Address
	To    Address
	Value *math.BigInt
	Data  []byte
}

func (t *Tx) String() string {
	return fmt.Sprintf("%s-[%.10s]-%d-Tx", t.TxBase.String(), t.Sender().String(), t.AccountNonce)
}

func SampleTx() *Tx {
	v, _ := math.NewBigIntFromString("-1234567890123456789012345678901234567890123456789012345678901234567890", 10)

	return &Tx{TxBase: TxBase{
		Height:       12,
		ParentsHash:  Hashes{HexToHash("0xCCDD"), HexToHash("0xEEFF")},
		Type:         TxBaseTypeNormal,
		AccountNonce: 234,
	},
		From:  HexToAddress("0x99"),
		To:    HexToAddress("0x88"),
		Value: v,
	}
}

func randomHash() Hash {
	v := math.NewBigInt(rand.Int63())
	sh := BigToHash(v.Value)
	h := sha256.New()
	data := []byte("456544546fhjsiodiruheswer8ih")
	h.Write(sh.Bytes[:])
	h.Write(data)
	sum := h.Sum(nil)
	sh.MustSetBytes(sum, PaddingRight)
	return sh
}

func RandomHash() Hash {
	return randomHash()
}

func randomAddress() Address {
	v := math.NewBigInt(rand.Int63())
	adr := BigToAddress(v.Value)
	h := sha256.New()
	data := []byte("abcd8342804fhddhfhisfdyr89")
	h.Write(adr.Bytes[:])
	h.Write(data)
	sum := h.Sum(nil)
	adr.MustSetBytes(sum[:20])
	return adr
}

func RandomTx() *Tx {
	return &Tx{TxBase: TxBase{
		Hash:         randomHash(),
		Height:       uint64(rand.Int63n(1000)),
		ParentsHash:  Hashes{randomHash(), randomHash()},
		Type:         TxBaseTypeNormal,
		AccountNonce: uint64(rand.Int63n(50000)),
		Weight:       uint64(rand.Int31n(2000)),
	},
		From:  randomAddress(),
		To:    randomAddress(),
		Value: math.NewBigInt(rand.Int63()),
	}
}

func (t *Tx) SignatureTargets() []byte {
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, t.AccountNonce))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.From.Bytes))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.To.Bytes))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Value.GetSigBytes()))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Data))

	return buf.Bytes()
}

func (t *Tx) Sender() Address {
	return t.From
}

func (t *Tx) GetValue() *math.BigInt {
	return t.Value
}

func (t *Tx) Parents() Hashes {
	return t.ParentsHash
}

func (t *Tx) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *Tx:
		if t.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (t *Tx) GetBase() *TxBase {
	return &t.TxBase
}

func (t *Tx) Dump() string {
	var phashes []string
	for _, p := range t.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	return fmt.Sprintf("hash %s pHash:[%s], from : %s , to :%s ,value : %s ,\n nonce : %d , signatute : %s, pubkey %s ,"+
		"height %d ,mined Nonce %v type %v weight %d", t.Hash.Hex(),
		strings.Join(phashes, " ,"), t.From.Hex(), t.To.Hex(), t.Value,
		t.AccountNonce, hexutil.Encode(t.Signature), hexutil.Encode(t.PublicKey), t.Height, t.MineNonce, t.Type, t.Weight)
}
func (t *Tx) RawTx() *RawTx {
	if t == nil {
		return nil
	}
	rawTx := &RawTx{
		TxBase: t.TxBase,
		To:     t.To,
		Value:  t.Value,
	}
	return rawTx
}

func (t Txs) String() string {
	var strs []string
	for _, v := range t {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (t Txs) ToRawTxs() RawTxs {
	if len(t) == 0 {
		return nil
	}
	var rawTxs []*RawTx
	for _, v := range t {
		rasTx := v.RawTx()
		rawTxs = append(rawTxs, rasTx)
	}
	return rawTxs
}
