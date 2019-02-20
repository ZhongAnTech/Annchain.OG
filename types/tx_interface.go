package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/annchain/OG/common/crypto/sha3"
	"github.com/tinylib/msgp/msgp"
)

//go:generate msgp
type TxBaseType uint16

const (
	TxBaseTypeNormal TxBaseType = iota
	TxBaseTypeSequencer
	TxBaseTypeCampaign
	TxBaseTypeTermChange
)

func (t TxBaseType) String() string {
	switch t {
	case TxBaseTypeNormal:
		return "TX"
	case TxBaseTypeSequencer:
		return "SQ"
	case TxBaseTypeCampaign:
		return "CP"
	case TxBaseTypeTermChange:
		return  "TC"
	default:
		return "NA"
	}
}

// Here indicates what fields should be concerned during hash calculation and signature generation
// |      |                   | Signature target |     NonceHash(Slow) |    TxHash(Fast) |
// |------|-------------------|------------------|---------------------|-----------------|
// | Base | ParentsHash       |                  |                     |               1 |
// | Base | Height            |                  |                     |                 |
// | Base | PublicKey         |                  |                   1 | 1 (nonce hash)  |
// | Base | Signature         |                  |                   1 | 1 (nonce hash)  |
// | Base | MinedNonce        |                  |                   1 | 1 (nonce hash)  |
// | Base | AccountNonce      |                1 |                     |                 |
// | Tx   | From              |                1 |                     |                 |
// | Tx   | To                |                1 |                     |                 |
// | Tx   | Value             |                1 |                     |                 |
// | Tx   | Data              |                1 |                     |                 |
// | Seq  | Id                |                1 |                     |                 |
// | Seq  | ContractHashOrder |                1 |                     |                 |

//msgp:tuple Txi
type Txi interface {
	// Implemented by TxBase
	GetType() TxBaseType
	GetHeight() uint64
	GetWeight() uint64
	GetTxHash() Hash
	GetNonce() uint64
	Parents() Hashes // Parents returns the hash of txs that it directly proves.
	SetHash(h Hash)
	String() string
	CalcTxHash() Hash    // TxHash returns a full tx hash (parents sealed by PoW stage 2)
	CalcMinedHash() Hash // NonceHash returns the part that needs to be considered in PoW stage 1.
	CalculateWeight(parents Txis) uint64

	// implemented by each tx type
	GetBase() *TxBase
	Sender() Address
	Dump() string             // For logger dump
	Compare(tx Txi) bool      // Compare compares two txs, return true if they are the same.
	SignatureTargets() []byte // SignatureTargets only returns the parts that needs to be signed by sender.

	// implemented by msgp
	DecodeMsg(dc *msgp.Reader) (err error)
	EncodeMsg(en *msgp.Writer) (err error)
	MarshalMsg(b []byte) (o []byte, err error)
	UnmarshalMsg(bts []byte) (o []byte, err error)
	Msgsize() (s int)
}

//msgp:tuple TxBase
type TxBase struct {
	Type         TxBaseType
	Hash         Hash
	ParentsHash  Hashes
	AccountNonce uint64
	Height       uint64
	PublicKey    []byte
	Signature    []byte
	MineNonce    uint64
	Weight       uint64
}

func (t *TxBase) GetType() TxBaseType {
	return t.Type
}

func (t *TxBase) GetHeight() uint64 {
	return t.Height
}

func (t *TxBase) GetWeight() uint64 {
	return t.Weight
}

func (t *TxBase) GetTxHash() Hash {
	return t.Hash
}

func (t *TxBase) GetNonce() uint64 {
	return t.AccountNonce
}

func (t *TxBase) Parents() Hashes {
	return t.ParentsHash
}

func (t *TxBase) SetHash(hash Hash) {
	t.Hash = hash
}

func (t *TxBase) String() string {
	return fmt.Sprintf("%d-[%.10s]-%d", t.Height, t.GetTxHash().Hex(), t.Weight)
}

func (t *TxBase) CalcTxHash() (hash Hash) {
	var buf bytes.Buffer

	for _, ancestor := range t.ParentsHash {
		panicIfError(binary.Write(&buf, binary.BigEndian, ancestor.Bytes))
	}
	// TODO do not use Height to calculate tx hash.
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Weight))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.CalcMinedHash().Bytes))

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:], PaddingNone)
	return
}

func (t *TxBase) CalcMinedHash() (hash Hash) {
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, t.PublicKey))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Signature))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.MineNonce))

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:], PaddingNone)
	return
}

//CalculateWeight  a core algorithm for tx sorting,
//a tx's weight must bigger than any of it's parent's weight  and bigger than any of it's elder transaction's
func (t *TxBase) CalculateWeight(parents Txis) uint64 {
	var maxWeight uint64
	for _, p := range parents {
		if p.GetWeight() > maxWeight {
			maxWeight = p.GetWeight()
		}
	}
	return maxWeight + 1
}

type Txis []Txi

func (t Txis) String() string {
	var strs []string
	for _, v := range t {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (t Txis) Len() int {
	return len(t)
}

func (t Txis) Less(i, j int) bool {
	if t[i].GetWeight() < t[j].GetWeight() {
		return true
	}
	if t[i].GetWeight() > t[j].GetWeight() {
		return false
	}
	if t[i].GetTxHash().Cmp(t[j].GetTxHash()) < 0 {
		return true
	}
	return false
}

func (t Txis) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
