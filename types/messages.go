package types

//go:generate msgp

//msgp:tuple MessageSyncRequest
type MessageSyncRequest struct {
	Hashes []Hash
}

//msgp:tuple MessageSyncResponse
type MessageSyncResponse struct {
	Txs []*Tx
	Sequencer []*Sequencer
}

type MessageNewTx struct{
	Tx *Tx
}

type MessageNewSequence struct {
	Sequencer *Sequencer
}