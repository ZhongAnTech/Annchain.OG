package annsensus

import (
	"fmt"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"golang.org/x/crypto/sha3"
)

//go:generate msgp

type TestMsg struct {
	Message     types.Message
	MessageType og.MessageType
	From        int
}

func (t *TestMsg) GetHash() types.Hash {
	//from := byte(t.From)
	data, err := t.Message.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	//data = append(data, from)
	h := sha3.New256()
	h.Write(data)
	b := h.Sum(nil)
	hash := types.Hash{}
	hash.MustSetBytes(b, types.PaddingNone)
	return hash
}

func (t TestMsg) String() {
	fmt.Sprintf("from %d, type %s, msg %s", t.From, t.MessageType, t.Message)
}
