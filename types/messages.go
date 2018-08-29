package types

//go:generate msgp

//msgp:tuple MessageSyncRequest
type MessageSyncRequest struct {
	Hashes []Hash
}

//msgp:tuple MessageSyncResponse
type MessageSyncResponse struct {
	Txs        []*Tx
	Sequencers []*Sequencer
}

//msgp:tuple MessageNewTx
type MessageNewTx struct{
	Tx *Tx
}

//msgp:tuple MessageNewSequence
type MessageNewSequence struct {
	Sequencer *Sequencer
}