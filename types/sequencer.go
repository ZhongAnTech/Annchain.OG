package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"github.com/annchain/OG/common/math"
)

//go:generate msgp
//msgp:tuple Sequencers

type Sequencer struct {
	// TODO: need more states in sequencer to differentiate multiple chains
	TxBase
	Id                uint64 `msgp:"id"`
	Issuer            Address
	ContractHashOrder []Hash `msgp:"contractHashOrder"`
}

func (t *Sequencer) String() string {
	return fmt.Sprintf("%s-[%.10s]-%d", t.TxBase.String(), t.Sender().String(), t.AccountNonce)
}

func SampleSequencer() *Sequencer {
	return &Sequencer{Id: 99,
		TxBase: TxBase{
			Height:       12,
			ParentsHash:  []Hash{HexToHash("0xCCDD"), HexToHash("0xEEFF")},
			Type:         TxBaseTypeSequencer,
			AccountNonce: 234,
		},
		Issuer: HexToAddress("0x33"),
		ContractHashOrder: []Hash{
			HexToHash("0x00667788"),
			HexToHash("0xAA667788"),
			HexToHash("0xBB667788"), // 20 bytes
		},
	}
}

func RandomSequencer() *Sequencer {
	return &Sequencer{TxBase: TxBase{
		Hash:         randomHash(),
		Height:       rand.Uint64(),
		ParentsHash:  []Hash{randomHash(), randomHash()},
		Type:         TxBaseTypeSequencer,
		AccountNonce: uint64(rand.Int63n(50000)),
	},
		Id:                rand.Uint64(),
		Issuer:            randomAddress(),
		ContractHashOrder: []Hash{randomHash(), randomHash(), randomHash()},
	}
}

func (t *Sequencer) SignatureTargets() []byte {
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, t.AccountNonce))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Issuer.Bytes))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Id))
	for _, orderHash := range t.ContractHashOrder {
		panicIfError(binary.Write(&buf, binary.BigEndian, orderHash.Bytes))
	}

	return buf.Bytes()
}

func (t *Sequencer) Sender() Address {
	return t.Issuer
}

func (t *Sequencer) GetValue() *math.BigInt {
	return math.NewBigInt(0)
}

func (t *Sequencer) Parents() []Hash {
	return t.ParentsHash
}

func (t *Sequencer) Number() uint64 {
	return t.Id
}

func (t *Sequencer) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *Sequencer:
		if t.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (t *Sequencer) GetBase() *TxBase {
	return &t.TxBase
}

func (t *Sequencer) GetHead() *SequencerHeader {
	return NewSequencerHead(t.GetTxHash(), t.Id)
}
