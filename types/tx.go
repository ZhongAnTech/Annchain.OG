package types

import (
	"bytes"
	"encoding/binary"
	"github.com/annchain/OG/common/math"
	"github.com/google/go-cmp/cmp"
	"fmt"
	"math/rand"
	"github.com/annchain/OG/common/crypto/sha3"
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

func SampleTx() *Tx {
	v, _ := math.NewBigIntFromString("-1234567890123456789012345678901234567890123456789012345678901234567890", 10)

	return &Tx{TxBase: TxBase{
		Height:       12,
		ParentsHash:  []Hash{HexToHash("0xCCDD"), HexToHash("0xEEFF")},
		Type:         TxBaseTypeNormal,
		AccountNonce: 234,
	},
		From: HexToAddress("0x99"),
		To: HexToAddress("0x88"),
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
		From: randomAddress(),
		To: randomAddress(),
		Value: math.NewBigInt(rand.Int63()),
	}
}

func (t *Tx) TxHash() (hash Hash) {
	var buf bytes.Buffer

	for _, ancestor := range t.ParentsHash {
		panicIfError(binary.Write(&buf, binary.BigEndian, ancestor.Bytes))
	}
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Height))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.NonceHash().Bytes))

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:])
	return
}

func (t *Tx) NonceHash() (hash Hash) {
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, t.PublicKey))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Signature))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.MineNonce))

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:])
	return
}

func (t *Tx) SignatureTargets() []byte {
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, t.AccountNonce))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.From.Bytes))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.To.Bytes))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Value.GetBytes()))

	return buf.Bytes()
}

func (t *Tx) Parents() []Hash {
	return t.ParentsHash
}

func (t *Tx) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *Tx:
		return cmp.Equal(t, tx)
	default:
		return false
	}
}
func (t *Tx) String() string {
	return fmt.Sprintf("[%s] %s From %s to %s", t.TxBase.String(), t.Value, t.From.Hex()[:10], t.To.Hex()[:10])
}

func (t *Tx) GetBase() *TxBase {
	return &t.TxBase
}
