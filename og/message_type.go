package og

import "github.com/annchain/OG/types"

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
	MessageType MessageType
	Message     []byte
	Hash        types.Hash    //
}
