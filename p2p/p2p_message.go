package p2p

//go:generate msgp
//msgp:tuple P2PMessage

const (
	MessageTypePing uint = iota
	MessageTypePong
)

type P2PMessage struct {
	MessageType uint
	Message     []byte
}
