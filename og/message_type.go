package og

import (
	"crypto/sha256"
	"github.com/annchain/OG/types"
)

//go:generate msgp
//msgp:tuple P2PMessage

const (
	OG31 = 31
	OG32 = 32
)

// ProtocolName is the official short name of the protocol used during capability negotiation.
var ProtocolName = "og"

// ProtocolVersions are the upported versions of the eth protocol (first is primary).
var ProtocolVersions = []uint{OG31, OG32}

// ProtocolLengths are the number of implemented message corresponding to different protocol versions.
var ProtocolLengths = []uint64{17, 8}

const ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message

type MessageType uint64

// og protocol message codes
const (
	// Protocol messages belonging to OG/31
	StatusMsg                      MessageType = iota
	MessageTypePing
	MessageTypePong
	MessageTypeFetchByHash
	MessageTypeFetchByHashResponse
	MessageTypeNewTx
	MessageTypeNewSequence

	// Protocol messages belonging to OG/32
	GetNodeDataMsg
	NodeDataMsg
	GetReceiptsMsg
)

func (mt MessageType) String() string {
	return []string{
		"StatusMsg", "MessageTypePing", "MessageTypePong", "MessageTypeFetchByHash", "MessageTypeFetchByHashResponse",
		"MessageTypeNewTx", "MessageTypeNewSequence", "GetNodeDataMsg", "NodeDataMsg", "GetReceiptsMsg",
	}[int(mt)]
}

type P2PMessage struct {
	MessageType     MessageType
	Message         []byte
	hash            types.Hash //inner use to avoid resend a message to the same peer
	needCheckRepeat bool
	SourceID		string		// the source that this messeage coming from
}

func (m *P2PMessage) calculateHash() {
	// TODO: implement hash for message
	h := sha256.New()
	h.Write(m.Message)
	sum := h.Sum(nil)
	m.hash.MustSetBytes(sum)
	return
}

func (m *P2PMessage) init() {
	if m.MessageType != MessageTypePing && m.MessageType != MessageTypePong {
		m.needCheckRepeat = true
		m.calculateHash()
	}
}

type errCode int

const (
	ErrMsgTooLarge             = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrProtocolVersionMismatch
	ErrNetworkIdMismatch
	ErrGenesisBlockMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSuspendedPeer
)

func (e errCode) String() string {
	return errorToString[int(e)]
}

// XXX change once legacy code is out
var errorToString = map[int]string{
	ErrMsgTooLarge:             "Message too long",
	ErrDecode:                  "Invalid message",
	ErrInvalidMsgCode:          "Invalid message code",
	ErrProtocolVersionMismatch: "Protocol version mismatch",
	ErrNetworkIdMismatch:       "NetworkId mismatch",
	ErrGenesisBlockMismatch:    "Genesis block mismatch",
	ErrNoStatusMsg:             "No status message",
	ErrExtraStatusMsg:          "Extra status message",
	ErrSuspendedPeer:           "Suspended peer",
}

// statusData is the network packet for the status message.
type StatusData struct {
	ProtocolVersion uint32
	NetworkId       uint64
	CurrentBlock    types.Hash
	GenesisBlock    types.Hash
}
