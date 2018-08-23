package types

import (
	"bytes"
	"encoding/binary"
	"golang.org/x/crypto/sha3"
)

//go:generate msgp
//msgp:tuple Sequencer

type Sequencer struct {
	Id                uint64     `msgp:"id"`
	TxBase
	ContractHashOrder []Hash     `msgp:"contractHashOrder"`
	Raws              [][32]byte `msgp:"Hashes"`
}

func (t *Sequencer) BlockHash() (hash Hash) {
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, t.Id))

	for _, ancestor := range t.ParentsHash {
		panicIfError(binary.Write(&buf, binary.BigEndian, ancestor.Bytes))
	}

	for _, orderHash := range t.ContractHashOrder {
		panicIfError(binary.Write(&buf, binary.BigEndian, orderHash.Bytes))
	}

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:])
	return
}
