package types

import (
	"encoding/json"
	"fmt"
)

type Block struct {
	Index        int64  `json:"index"`
	PreviousHash string `json:"previousHash"`
	Timestamp    int64  `json:"timestamp"`
	Data         string `json:"data"`
	Hash         string `json:"hash"`

	PrevBlock *Block
	NextBlock *Block
}

func (b *Block) Bytes() []byte {
	r, _ := json.Marshal(b)
	return r
}
func (b *Block) String() string {
	return fmt.Sprintf("index: %d,previousHash:%s,timestamp:%d,data:%s,hash:%s", b.Index, b.PreviousHash, b.Timestamp, b.Data, b.Hash)
}
