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
type MessageNewTx struct {
	Tx *Tx
}

//msgp:tuple MessageNewSequence
type MessageNewSequence struct {
	Sequencer *Sequencer
}

//msgp:tuple MessageNewTxs
type MessageNewTxs struct {
	Txs []*Tx
}

//msgp:tuple MessageTxsRequest
type MessageTxsRequest struct {
	Hashes  []Hash
	SeqHash *Hash
	Id      uint64
}

//msgp:tuple MessageTxsResponse
type MessageTxsResponse struct {
	Txs       []*Tx
	Sequencer *Sequencer
}

// getBlockHeadersData represents a block header query.
//msgp:tuple MessageHeaderRequest
type MessageHeaderRequest struct {
	Origin  HashOrNumber // Block from which to retrieve headers
	Amount  uint64       // Maximum number of headers to retrieve
	Skip    uint64       // Blocks to skip between consecutive headers
	Reverse bool         // Query direction (false = rising towards latest, true = falling towards genesis)
}

// hashOrNumber is a combined field for specifying an origin block.
//msgp:tuple HashOrNumber
type HashOrNumber struct {
	Hash   Hash   // Block hash from which to retrieve headers (excludes Number)
	Number uint64 // Block hash from which to retrieve headers (excludes Hash)
}

//msgp:tuple MessageSequencerHeader
type MessageSequencerHeader struct {
	Hash   *Hash
	Number uint64
}

//msgp:tuple MessageHeaderResponse
type MessageHeaderResponse struct {
	Sequencers []*Sequencer
}

//msgp:tuple MessageBodiesRequest
type MessageBodiesRequest struct {
	SeqHashes []Hash
}

//msgp:tuple MessageBodiesResponse
type MessageBodiesResponse struct {
	Bodies []RawData
}

type RawData []byte
