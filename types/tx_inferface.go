package types

//go:generate msgp
//cccmsgp:tuple TxBase

type Txi interface {
	// Hash returns a tx hash
	Hash() Hash

	// Parents returns the hash of txs that it directly proves.
	Parents() []Hash

	// Compare compares two txs, return true if they are the same.
	Compare(Txi) bool
}

type TxBase struct {
	Type          int    `msgp:"type"`
	ParentsHash   []Hash `msgp:"parentHash"`
	SequenceNonce uint64 `msgp:"sequenceNonce"`
	Height        uint64 `msgp:"height"`
}