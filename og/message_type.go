package og

import (
	"crypto/sha256"
	"github.com/annchain/OG/types"
)

//go:generate msgp
//msgp:tuple P2PMessage

type MessageType uint

const (
	MessageTypePing MessageType = iota
	MessageTypePong
	MessageTypeFetchByHash
	MessageTypeFetchByHashResponse
	MessageTypeNewTx
	MessageTypeNewSequence
)

type P2PMessage struct {
	MessageType     MessageType
	Message         []byte
	hash            types.Hash //inner use to avoid resend a message to the same peer
	needCheckRepeat bool
}

func (m *P2PMessage) calculateHash() {
	// TODO: implement hash for message
	h := sha256.New()
	h.Write(m.Message)
	sum := h.Sum(nil)
	m.hash.MustSetBytes(sum)
	return
}
