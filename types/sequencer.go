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

func (t *Sequencer) MinedHash() (hash Hash) {
	var buf bytes.Buffer

	for _, ancestor := range t.ParentsHash {
		panicIfError(binary.Write(&buf, binary.BigEndian, ancestor.Bytes))
	}

	panicIfError(binary.Write(&buf, binary.BigEndian, t.AccountNonce))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Height))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.PublicKey))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Signature))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.MineNonce))

	panicIfError(binary.Write(&buf, binary.BigEndian, t.Id))
	for _, orderHash := range t.ContractHashOrder {
		panicIfError(binary.Write(&buf, binary.BigEndian, orderHash.Bytes))
	}

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:])
	return
}

func (t *Sequencer) StructureHash() (hash Hash) {
	var buf bytes.Buffer
	for _, ancestor := range t.ParentsHash {
		panicIfError(binary.Write(&buf, binary.BigEndian, ancestor.Bytes))
	}

	panicIfError(binary.Write(&buf, binary.BigEndian, t.Height))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.PublicKey))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Signature))

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:])
	return
}

func (t *Sequencer) SignatureTargets() []byte {
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, t.AccountNonce))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Id))
	for _, orderHash := range t.ContractHashOrder {
		panicIfError(binary.Write(&buf, binary.BigEndian, orderHash.Bytes))
	}

	return buf.Bytes()
}


func (t *Sequencer) Parents() []Hash {
	return t.ParentsHash
}

func (t *Sequencer) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *Sequencer:
		return cmp.Equal(t, tx)
	default:
		return false
	}
}

func (t *Sequencer) String() string {
	var hashes []string
	for _, v := range t.ContractHashOrder {
		hashes = append(hashes, v.Hex()[0:10])
	}

	return fmt.Sprintf("[%s] %d Hashes %s", t.TxBase.String(), t.Id, strings.Join(hashes, ","))
}

func (t *Sequencer) GetBase() TxBase{
	return t.TxBase
}