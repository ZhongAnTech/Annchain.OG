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

//msgp:tuple Txi
type Txi interface {
	// Hash returns a tx hash
	Hash() Hash

	// Parents returns the hash of txs that it directly proves.
	Parents() []Hash
	GetType() TxBaseType
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

func (t *TxBase) String() string {
	var hashes []string
	for _, v := range t.ParentsHash {
		hashes = append(hashes, v.Hex()[0:10])
	}

	return fmt.Sprintf("%s %s Parent [%s]", t.Type.String(), t.Hash.Hex()[:10], strings.Join(hashes, ","))
}
