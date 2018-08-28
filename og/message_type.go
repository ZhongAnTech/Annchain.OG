package og

//go:generate msgp
//msgp:tuple P2PMessage

type MessageType uint

const (
	MessageTypePing MessageType = iota
	MessageTypePong
	MessageTypeFetchByHash
	MessageTypeFetchByHashResponse
)

type P2PMessage struct {
	MessageType MessageType
	Message     []byte
}
