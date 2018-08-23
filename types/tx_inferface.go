package types

//go:generate msgp
//msgp:tuple TxBase

type Txi interface {
	// Hash returns a tx hash
	Hash() Hash

	// Parents returns the hash of txs that it directly proves.
	Parents() []Hash
}
type TxBase struct {
	Type          int    `msgp:"type"`
	ParentsHash   []Hash `msgp:"parentHash"`
	SequenceNonce uint64 `msgp:"sequenceNonce"`
	Height        uint64 `msgp:"height"`
}
