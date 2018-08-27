package types

import (
	"bytes"
	"encoding/binary"
	"golang.org/x/crypto/sha3"
)

//go:generate msgp
//msgp:tuple Sequencer

type Sequencer struct {
	TxBase
	Id                uint64     `msgp:"id"`
	ContractHashOrder []Hash     `msgp:"contractHashOrder"`
	Raws              [][32]byte `msgp:"Hashes"`
}

func SampleSequencer() *Sequencer {
	return &Sequencer{Id: 99,
		TxBase: TxBase{
			Height:        12,
			ParentsHash:   []Hash{HexToHash("0xCCDD"), HexToHash("0xEEFF"),},
			Type:          1,
			SequenceNonce: 234,
		},
		ContractHashOrder: []Hash{
			HexToHash("0x00667788"),
			HexToHash("0xAA667788"),
			HexToHash("0xBB667788"), // 20 bytes
		},
	}
}

func (t *Sequencer) Hash() (hash Hash) {
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

