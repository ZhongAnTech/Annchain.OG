package types

import (
	"golang.org/x/crypto/sha3"
	"bytes"
	"encoding/binary"
	"github.com/annchain/OG/common/math"
	"fmt"
)

//go:generate msgp
//msgp:tuple Tx

type Tx struct {
	TxBase
	From  Address
	To    Address
	Value *math.BigInt
}

func SampleTx() *Tx {
	v, _ := math.NewBigIntFromString("-1234567890123456789012345678901234567890123456789012345678901234567890", 10)

	return &Tx{TxBase: TxBase{
		Height:        12,
		ParentsHash:   []Hash{HexToHash("0xCCDD"), HexToHash("0xEEFF"),},
		Type:          TxBaseTypeNormal,
		SequenceNonce: 234,
	},
		From: HexToAddress("0x99"),
		To: HexToAddress("0x88"),
		Value: v,
	}
}

func (t *Tx) Hash() (hash Hash) {
	var buf bytes.Buffer

	err := binary.Write(&buf, binary.BigEndian, t.Height)
	panicIfError(err)

	for _, ancestor := range t.ParentsHash {
		panicIfError(binary.Write(&buf, binary.BigEndian, ancestor.Bytes))
	}
	panicIfError(binary.Write(&buf, binary.BigEndian, t.From.Bytes))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.To.Bytes))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Value.GetBytes()))

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:])
	return
}

func (t *Tx) String() string {
	return fmt.Sprintf("[%s] %s From %s to %s", t.TxBase.String(), t.Value, t.From.Hex()[:10], t.To.Hex()[:10])
}
