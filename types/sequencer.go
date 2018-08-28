package types

import (
	"bytes"
	"encoding/binary"
	"golang.org/x/crypto/sha3"
	"github.com/google/go-cmp/cmp"
)

//go:generate msgp
//cccmsgp:tuple Sequencer

type Sequencer struct {
	Id uint64 `msgp:"id"`
	TxBase
	ContractHashOrder []Hash `msgp:"contractHashOrder"`
}

func (seq *Sequencer) Hash() (hash Hash) {
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, seq.Id))

	for _, ancestor := range seq.ParentsHash {
		panicIfError(binary.Write(&buf, binary.BigEndian, ancestor.Bytes))
	}

	for _, orderHash := range seq.ContractHashOrder {
		panicIfError(binary.Write(&buf, binary.BigEndian, orderHash.Bytes))
	}

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:])
	return
}

func (seq *Sequencer) Parents() []Hash {
	return seq.ParentsHash
}

func (seq *Sequencer) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *Sequencer:
		return cmp.Equal(seq, tx)
	default:
		return false
	}
}
