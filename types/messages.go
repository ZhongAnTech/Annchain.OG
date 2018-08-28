package types

//go:generate msgp

//msgp:tuple MessageSyncRequest
type MessageSyncRequest struct {
	Hashes []Hash
}

//msgp:tuple MessageSyncResponse
type MessageSyncResponse struct {
	Txs []Txi
}
