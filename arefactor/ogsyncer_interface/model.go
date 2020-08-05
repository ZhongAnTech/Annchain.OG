package ogsyncer_interface

//go:generate msgp

type OgSyncType int

const (
	OgSyncTypeLatestHeight OgSyncType = iota
	OgSyncTypeByHashes
	OgSyncTypeBlockByHeight
	OgSyncTypeBlockByHash
)

//msgp OgSyncLatestHeightRequest
type OgSyncLatestHeightRequest struct {
	MyHeight int64
}

//msgp OgSyncLatestHeightResponse
type OgSyncLatestHeightResponse struct {
	MyHeight int64
}

//msgp OgSyncByHashesRequest
type OgSyncByHashesRequest struct {
	Hashes [][]byte
}

//msgp OgSyncByHashesResponse
type OgSyncByHashesResponse struct {
	Sequencers []MessageContentSequencer
	Txs        []MessageContentTx
	Ints       []MessageContentInt
}

//msgp OgSyncBlockByHeightRequest
type OgSyncBlockByHeightRequest struct {
	Height int64
	Offset int
}

//msgp OgSyncBlockByHeightResponse
type OgSyncBlockByHeightResponse struct {
	HasMore    bool
	Sequencers MessageContentSequencer
	Ints       MessageContentInt
	Txs        []MessageContentTx
}

//msgp OgSyncBlockByHashRequest
type OgSyncBlockByHashRequest struct {
	Hash   []byte
	Offset int
}

//msgp OgSyncBlockByHashResponse
type OgSyncBlockByHashResponse struct {
	HasMore    bool
	Sequencers MessageContentSequencer
	Ints       MessageContentInt
	Txs        []MessageContentTx
}
