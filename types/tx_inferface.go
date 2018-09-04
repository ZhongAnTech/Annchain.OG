package types

import (
	"github.com/tinylib/msgp/msgp"
	"fmt"
	"strings"
)

//go:generate msgp
type TxBaseType uint

const (
	TxBaseTypeNormal    TxBaseType = iota
	TxBaseTypeSequencer
)

func (t TxBaseType) String() string {
	switch t {
	case TxBaseTypeNormal:
		return "TX"
	case TxBaseTypeSequencer:
		return "SQ"
	default:
		return "NA"
	}
}

// Here indicates what fields should be concerned during hash calculation and signature generation
// |      |                   | Signature target | StructureHash(Slow) | MinedHash(Fast) |
// |------|-------------------|------------------|---------------------|-----------------|
// | Base | ParentsHash       |                  |                   1 |               1 |
// | Base | AccountNonce      |                1 |                     |               1 |
// | Base | Height            |                  |                   1 |               1 |
// | Base | PublicKey         |                  |                   1 |               1 |
// | Base | Signature         |                  |                   1 |               1 |
// | Base | MineNonce         |                  |                     |               1 |
// | Tx   | From              |                1 |                     |               1 |
// | Tx   | To                |                1 |                     |               1 |
// | Tx   | Value             |                1 |                     |               1 |
// | Seq  | Id                |                1 |                     |               1 |
// | Seq  | ContractHashOrder |                1 |                     |               1 |

//msgp:tuple Txi
type Txi interface {
	MinedHash() Hash          // MinedHash returns a full tx hash (MineNonce sealed)
	StructureHash() Hash      // StructureHash returns the part that needs to be copnsidered in building the structure of dag.
	SignatureTargets() []byte // SignatureTargets only returns the parts that needs to be signed by sender.
	Parents() []Hash          // Parents returns the hash of txs that it directly proves.

	Compare(tx Txi) bool // Compare compares two txs, return true if they are the same.

	GetType() TxBaseType
	GetBase() *TxBase
	String() string
	SetMineNonce(mineNonce uint64)

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
	ParentsHash  []Hash
	AccountNonce uint64
	Height       uint64
	PublicKey    []byte
	Signature    []byte
	MineNonce    uint64
}

func (t *TxBase) GetType() TxBaseType {
	return t.Type
}

func (t *TxBase) Parents() []Hash {
	return t.ParentsHash
}

func (t *TxBase) SetHash(hash Hash) {
	t.Hash = hash
}
func (t *TxBase) SetMineNonce(mineNonce uint64) {
	t.MineNonce = mineNonce
}

func (t *TxBase) String() string {
	var hashes []string
	for _, v := range t.ParentsHash {
		hashes = append(hashes, v.Hex()[0:10])
	}

	return fmt.Sprintf("%s %s Parent [%s]", t.Type.String(), t.Hash.Hex()[:10], strings.Join(hashes, ","))
}
