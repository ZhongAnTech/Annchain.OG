package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/annchain/OG/common/math"
	"math/rand"
)

//go:generate msgp
//msgp:tuple Tx

type Txs []*Tx

type Tx struct {
	TxBase
	From  Address
	To    Address
	Value *math.BigInt
}

func (t *Tx) String() string {
	return fmt.Sprintf("%s-[%.10s]-%d-Tx", t.TxBase.String(), t.Sender().String(), t.AccountNonce)
}

func SampleTx() *Tx {
	v, _ := math.NewBigIntFromString("-1234567890123456789012345678901234567890123456789012345678901234567890", 10)

	return &Tx{TxBase: TxBase{
		Height:       12,
		ParentsHash:  []Hash{HexToHash("0xCCDD"), HexToHash("0xEEFF")},
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
	return BigToHash(v.Value)
}
func randomAddress() Address {
	v := math.NewBigInt(rand.Int63())
	return BigToAddress(v.Value)
}

func RandomTx() *Tx {
	return &Tx{TxBase: TxBase{
		Hash:         randomHash(),
		Height:       rand.Uint64(),
		ParentsHash:  []Hash{randomHash(), randomHash()},
		Type:         TxBaseTypeNormal,
		AccountNonce: uint64(rand.Int63n(50000)),
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
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Value.GetBytes()))

	return buf.Bytes()
}

func (t *Tx) Sender() Address {
	return t.From
}

func (t *Tx) GetValue() *math.BigInt {
	return t.Value
}

func (t *Tx) Parents() []Hash {
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
