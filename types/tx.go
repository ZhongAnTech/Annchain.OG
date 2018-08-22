package types

import (
	"golang.org/x/crypto/sha3"
	"bytes"
	"encoding/binary"
	"github.com/annchain/OG/common/math"
)

//go:generate msgp
//msgp:tuple Tx

type Tx struct {
	TxBase
	From          Address
	To            Address
	Value         *math.BigInt
}

func (t *Tx) BlockHash() (hash Hash) {
	var buf bytes.Buffer

	err := binary.Write(&buf, binary.LittleEndian, t.Height)
	panicIfError(err)

	for _, ancestor := range t.ParentsHash {
		err = binary.Write(&buf, binary.LittleEndian, ancestor)
		panicIfError(err)
	}

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:])
	return
}
