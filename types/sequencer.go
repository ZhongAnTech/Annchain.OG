package types

import (
	"bytes"
	"encoding/binary"
	"golang.org/x/crypto/sha3"
	"github.com/google/go-cmp/cmp"
	"fmt"
	"strings"
)

//go:generate msgp
//msgp:tuple Sequencers

type Sequencer struct {
	TxBase
	Id                uint64 `msgp:"id"`
	ContractHashOrder []Hash `msgp:"contractHashOrder"`
}

func SampleSequencer() *Sequencer {
	return &Sequencer{Id: 99,
		TxBase: TxBase{
			Height:       12,
			ParentsHash:  []Hash{HexToHash("0xCCDD"), HexToHash("0xEEFF"),},
			Type:         TxBaseTypeSequencer,
			AccountNonce: 234,
		},
		ContractHashOrder: []Hash{
			HexToHash("0x00667788"),
			HexToHash("0xAA667788"),
			HexToHash("0xBB667788"), // 20 bytes
		},
	}
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

	panicIfError(binary.Write(&buf, binary.BigEndian, seq.MineNonce))

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

func (seq *Sequencer) String() string {
	var hashes []string
	for _, v := range seq.ContractHashOrder {
		hashes = append(hashes, v.Hex()[0:10])
	}

	return fmt.Sprintf("[%s] %d Hashes %s", seq.TxBase.String(), seq.Id, strings.Join(hashes, ","))
}

func (seq *Sequencer) GetBase() TxBase{
	return seq.TxBase
}