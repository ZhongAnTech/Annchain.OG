package og

import (
	"crypto/sha256"
	"fmt"
	"github.com/annchain/OG/types"
	"sync/atomic"
)

//go:generate msgp
//msgp:tuple P2PMessage

const (
	OG31 = 31
	OG32 = 32
)

// ProtocolName is the official short name of the protocol used during capability negotiation.
var ProtocolName = "og"

// ProtocolVersions are the supported versions of the og protocol (first is primary).
var ProtocolVersions = []uint{OG32, OG31}

// ProtocolLengths are the number of implemented message corresponding to different protocol versions.
var ProtocolLengths = []MessageType{21, 18}

const ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message

type MessageType uint16

//global msg counter , generate global msg request id
var MsgCounter *MessageCounter

// og protocol message codes
const (
	// Protocol messages belonging to OG/31
	StatusMsg MessageType = iota
	MessageTypePing
	MessageTypePong
	MessageTypeFetchByHashRequest
	MessageTypeFetchByHashResponse
	MessageTypeNewTx
	MessageTypeNewSequencer
	MessageTypeNewTxs
	MessageTypeSequencerHeader

	MessageTypeBodiesRequest
	MessageTypeBodiesResponse

	MessageTypeTxsRequest
	MessageTypeTxsResponse
	MessageTypeHeaderRequest
	MessageTypeHeaderResponse

	//for optimizing network
	MessageTypeGetMsg
	MessageTypeDuplicate
	MessageTypeControl

	// Protocol messages belonging to OG/32

	GetNodeDataMsg
	NodeDataMsg
	GetReceiptsMsg
)

type SendingType uint8

const (
	sendingTypeBroacast SendingType = iota
	sendingTypeMulticast
	sendingTypeMulticastToSource
	sendingTypeBroacastWithFilter
	sendingTypeBroacastWithLink
)

func (mt MessageType) String() string {
	return []string{
		"StatusMsg", "MessageTypePing", "MessageTypePong", "MessageTypeFetchByHashRequest", "MessageTypeFetchByHashResponse",
		"MessageTypeNewTx", "MessageTypeNewSequencer", "MessageTypeNewTxs", "MessageTypeLatestSequencer",
		"MessageTypeBodiesRequest", "MessageTypeBodiesResponse", "MessageTypeTxsRequest",
		"MessageTypeTxsResponse", "MessageTypeHeaderRequest", "MessageTypeHeaderResponse",
		"MessageTypeGetMsg", "MessageTypeDuplicate", "MessageTypeControl",
		"GetNodeDataMsg", "NodeDataMsg", "GetReceiptsMsg",
	}[int(mt)]
}

type P2PMessage struct {
	MessageType MessageType
	data        []byte
	hash        types.Hash  //inner use to avoid resend a message to the same peer
	SourceID    string      // the source that this message  coming from ,fou outgoing it is nil
	sendingType SendingType //sending type
	Version     int         // peer version.
	Message     types.Message
	SourceHash  *types.Hash
}

func (m *P2PMessage) calculateHash() {

	// for txs,or response msg , even if  source peer id is different ,they were duplicated txs
	//for request ,if source id is different they were different  msg ,don't drop it
	//if we dropped header response because of duplicate , header request will time out
	data := m.data
	var hash *types.Hash
	if m.MessageType == MessageTypeBodiesRequest || m.MessageType == MessageTypeFetchByHashRequest ||
		m.MessageType == MessageTypeTxsRequest || m.MessageType == MessageTypeHeaderRequest ||
		m.MessageType == MessageTypeSequencerHeader || m.MessageType == MessageTypeHeaderResponse ||
		m.MessageType == MessageTypeBodiesResponse || m.MessageType == MessageTypeGetMsg ||
		m.MessageType == MessageTypeDuplicate {
		data = append(data, []byte(m.SourceID+"hi")...)
	} else if m.MessageType == MessageTypeNewTx {
		msg := m.Message.(*types.MessageNewTx)
		hash = msg.GetHash()
		if hash != nil {
			m.hash = *hash
			return
		} else {
			msgLog.Error("hash is nil , programmer error")
		}
	} else if m.MessageType == MessageTypeNewSequencer {
		msg := m.Message.(*types.MessageNewSequencer)
		hash = msg.GetHash()
		if hash != nil {
			m.hash = *hash
			return
		} else {
			msgLog.Error("hash is nil , programmer error")
		}
	}

	h := sha256.New()
	h.Write(data)
	sum := h.Sum(nil)
	m.hash.MustSetBytes(sum)
	return
}

type errCode int

const (
	ErrMsgTooLarge = iota
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
	CurrentId       uint64
}

func (s *StatusData) String() string {
	return fmt.Sprintf("ProtocolVersion  %d   NetworkId %d  CurrentBlock %s  GenesisBlock %s  CurrentId %d",
		s.ProtocolVersion, s.NetworkId, s.CurrentBlock, s.GenesisBlock, s.CurrentId)
}

type MessageCounter struct {
	requestId uint32
}

//get current request id
func (m *MessageCounter) Get() uint32 {
	if m.requestId > uint32(1<<30) {
		atomic.StoreUint32(&m.requestId, 10)
	}
	return atomic.AddUint32(&m.requestId, 1)
}

func MsgCountInit() {
	MsgCounter = &MessageCounter{
		requestId: 1,
	}
}

func (p *P2PMessage) GetMessage() error {
	switch p.MessageType {
	case MessageTypePing:
		p.Message = &types.MessagePing{}
	case MessageTypePong:
		p.Message = &types.MessagePong{}
	case MessageTypeFetchByHashRequest:
		p.Message = &types.MessageSyncRequest{}
	case MessageTypeFetchByHashResponse:
		p.Message = &types.MessageSyncResponse{}
	case MessageTypeNewTx:
		p.Message = &types.MessageNewTx{}
	case MessageTypeNewSequencer:
		p.Message = &types.MessageNewSequencer{}
	case MessageTypeNewTxs:
		p.Message = &types.MessageNewTxs{}
	case MessageTypeSequencerHeader:
		p.Message = &types.MessageSequencerHeader{}

	case MessageTypeBodiesRequest:
		p.Message = &types.MessageBodiesRequest{}
	case MessageTypeBodiesResponse:
		p.Message = &types.MessageBodiesResponse{}

	case MessageTypeTxsRequest:
		p.Message = &types.MessageTxsRequest{}
	case MessageTypeTxsResponse:
		p.Message = &types.MessageTxsResponse{}
	case MessageTypeHeaderRequest:
		p.Message = &types.MessageHeaderRequest{}
	case MessageTypeHeaderResponse:
		p.Message = &types.MessageHeaderResponse{}
	case MessageTypeDuplicate:
		p.Message = &types.MessageDuplicate{}
	case MessageTypeGetMsg:
		p.Message = &types.MessageGetMsg{}
	case MessageTypeControl:
		p.Message = &types.MessageControl{}
	default:
		return fmt.Errorf("unkown mssage type %v ", p.MessageType)
	}
	return nil
}
