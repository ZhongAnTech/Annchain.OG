package types

import "github.com/tinylib/msgp/msgp"

//go:generate msgp
type TxBaseType uint

const (
	TxBaseTypeNormal    TxBaseType = iota
	TxBaseTypeSequencer
)

//msgp:tuple Txi
type Txi interface {
	// Hash returns a tx hash
	Hash() Hash

	// Parents returns the hash of txs that it directly proves.
	Parents() []Hash
	GetType() TxBaseType

	DecodeMsg(dc *msgp.Reader) (err error)
	EncodeMsg(en *msgp.Writer) (err error)
	MarshalMsg(b []byte) (o []byte, err error)
	UnmarshalMsg(bts []byte) (o []byte, err error)
	Msgsize() (s int)

}

//msgp:tuple TxBase
type TxBase struct {
	Type          TxBaseType `msgp:"type"`
	ParentsHash   []Hash     `msgp:"parentHash"`
	SequenceNonce uint64     `msgp:"sequenceNonce"`
	Height        uint64     `msgp:"height"`
}

func (t *TxBase) GetType() TxBaseType {
	return t.Type
}

func (t *TxBase) Parents() []Hash{
	return t.ParentsHash
}
