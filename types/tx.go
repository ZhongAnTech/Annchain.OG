package types

import (
	"bytes"
	"encoding/binary"
	"github.com/annchain/OG/common/math"
	"golang.org/x/crypto/sha3"
)

//go:generate msgp
//cccmsgp:tuple Tx

type Txs []*Tx

type Tx struct {
	TxBase
	From  Address
	To    Address
	Value *math.BigInt
}

func (t *Tx) BlockHash() (hash Hash) {
	var buf bytes.Buffer

	err := binary.Write(&buf, binary.LittleEndian, t.Height)
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

func (t Tx) Hash() (hash Hash) {
	return
}
