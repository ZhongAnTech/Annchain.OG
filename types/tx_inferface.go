package types

import (
	"github.com/tinylib/msgp/msgp"
	"fmt"
	"strings"
	"bytes"
	"encoding/binary"
	"github.com/annchain/OG/common/crypto/sha3"
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
// |      |                   | Signature target |     NonceHash(Slow) |    TxHash(Fast) |
// |------|-------------------|------------------|---------------------|-----------------|
// | Base | ParentsHash       |                  |                     |               1 |
// | Base | Height            |                  |                     |               1 |
// | Base | PublicKey         |                  |                   1 | 1 (nonce hash)  |
// | Base | Signature         |                  |                   1 | 1 (nonce hash)  |
// | Base | MinedNonce        |                  |                   1 | 1 (nonce hash)  |
// | Base | AccountNonce      |                1 |                     |                 |
// | Tx   | From              |                1 |                     |                 |
// | Tx   | To                |                1 |                     |                 |
// | Tx   | Value             |                1 |                     |                 |
// | Seq  | Id                |                1 |                     |                 |
// | Seq  | ContractHashOrder |                1 |                     |                 |

//msgp:tuple Txi
type Txi interface {
	CalcTxHash() Hash         // TxHash returns a full tx hash (parents sealed by PoW stage 2)
	CalcNonceHash() Hash      // NonceHash returns the part that needs to be considered in PoW stage 1.
	SignatureTargets() []byte // SignatureTargets only returns the parts that needs to be signed by sender.
	Parents() []Hash          // Parents returns the hash of txs that it directly proves.

	Compare(tx Txi) bool // Compare compares two txs, return true if they are the same.

	GetType() TxBaseType
	GetBase() *TxBase
	GetTxHash() Hash
	String() string

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

func (t *TxBase) GetTxHash() Hash {
	return t.Hash
}

func (t *TxBase) Parents() []Hash {
	return t.ParentsHash
}

func (t *TxBase) SetHash(hash Hash) {
	t.Hash = hash
}

func (t *TxBase) String() string {
	var hashes []string
	for _, v := range t.ParentsHash {
		hashes = append(hashes, v.Hex()[0:10])
	}

	return fmt.Sprintf("%s %s Parent [%s]", t.Type.String(), t.Hash.Hex()[:10], strings.Join(hashes, ","))
}

func (t *TxBase) CalcTxHash() (hash Hash) {
	var buf bytes.Buffer

	for _, ancestor := range t.ParentsHash {
		panicIfError(binary.Write(&buf, binary.BigEndian, ancestor.Bytes))
	}
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Height))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.CalcNonceHash().Bytes))

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:])
	return
}

func (t *TxBase) CalcNonceHash() (hash Hash) {
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, t.PublicKey))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Signature))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.MineNonce))

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:])
	return
}
