package og

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types"
	"github.com/pkg/errors"
	"sync/atomic"
)

//go:generate msgp
//msgp:tuple p2PMessage

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

//global msg counter , generate global msg request id
var MsgCounter *MessageCounter

type MessageType uint16

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
		"MessageTypeNewTx", "MessageTypeNewSequencer", "MessageTypeNewTxs", "MessageTypeSequencerHeader",
		"MessageTypeBodiesRequest", "MessageTypeBodiesResponse", "MessageTypeTxsRequest",
		"MessageTypeTxsResponse", "MessageTypeHeaderRequest", "MessageTypeHeaderResponse",
		"MessageTypeGetMsg", "MessageTypeDuplicate", "MessageTypeControl",
		"GetNodeDataMsg", "NodeDataMsg", "GetReceiptsMsg",
	}[int(mt)]
}

func (mt MessageType) Code() p2p.MsgCodeType {
	return p2p.MsgCodeType(mt)
}

type p2PMessage struct {
	messageType  MessageType
	data         []byte
	hash         *types.Hash //inner use to avoid resend a message to the same peer
	sourceID     string      // the source that this message  coming from , outgoing if it is nil
	sendingType  SendingType //sending type
	version      int         // peer version.
	message      types.Message
	sourceHash   *types.Hash
	marshalState bool
}

func (m *p2PMessage) calculateHash() {
	// for txs,or response msg , even if  source peer id is different ,they were duplicated txs
	//for request ,if source id is different they were different  msg ,don't drop it
	//if we dropped header response because of duplicate , header request will time out
	if len(m.data) == 0 {
		msgLog.Error("nil data to calculate hash")
	}
	data := m.data
	var hash *types.Hash
	switch m.messageType {
	case MessageTypeNewTx:
		msg := m.message.(*types.MessageNewTx)
		hash = msg.GetHash()
		var msgHash types.Hash
		msgHash = *hash
		m.hash = &msgHash
		return
	case MessageTypeControl:
		msg := m.message.(*types.MessageControl)
		var msgHash types.Hash
		msgHash = *msg.Hash
		m.hash = &msgHash
		return

	case MessageTypeNewSequencer:
		msg := m.message.(*types.MessageNewSequencer)
		hash = msg.GetHash()
		var msgHash types.Hash
		msgHash = *hash
		m.hash = &msgHash
		return
	case MessageTypeTxsRequest:
		data = append(data, []byte(m.sourceID+"txs")...)
	case MessageTypeBodiesRequest:
		data = append(data, []byte(m.sourceID+"bq")...)
	case MessageTypeFetchByHashRequest:
		data = append(data, []byte(m.sourceID+"fe")...)
	case MessageTypeHeaderRequest:
		data = append(data, []byte(m.sourceID+"hq")...)
	case MessageTypeHeaderResponse:
		data = append(data, []byte(m.sourceID+"hp")...)
	case MessageTypeBodiesResponse:
		data = append(data, []byte(m.sourceID+"bp")...)
	case MessageTypeSequencerHeader:
		data = append(data, []byte(m.sourceID+"sq")...)
	case MessageTypeGetMsg:
		data = append(data, []byte(m.sourceID+"gm")...)
	default:
	}
	h := sha256.New()
	h.Write(data)
	sum := h.Sum(nil)
	m.hash = &types.Hash{}
	m.hash.MustSetBytes(sum, types.PaddingNone)
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

func (p *p2PMessage) GetMessage() error {
	switch p.messageType {
	case MessageTypePing:
		p.message = &types.MessagePing{}
	case MessageTypePong:
		p.message = &types.MessagePong{}
	case MessageTypeFetchByHashRequest:
		p.message = &types.MessageSyncRequest{}
	case MessageTypeFetchByHashResponse:
		p.message = &types.MessageSyncResponse{}
	case MessageTypeNewTx:
		p.message = &types.MessageNewTx{}
	case MessageTypeNewSequencer:
		p.message = &types.MessageNewSequencer{}
	case MessageTypeNewTxs:
		p.message = &types.MessageNewTxs{}
	case MessageTypeSequencerHeader:
		p.message = &types.MessageSequencerHeader{}

	case MessageTypeBodiesRequest:
		p.message = &types.MessageBodiesRequest{}
	case MessageTypeBodiesResponse:
		p.message = &types.MessageBodiesResponse{}

	case MessageTypeTxsRequest:
		p.message = &types.MessageTxsRequest{}
	case MessageTypeTxsResponse:
		p.message = &types.MessageTxsResponse{}
	case MessageTypeHeaderRequest:
		p.message = &types.MessageHeaderRequest{}
	case MessageTypeHeaderResponse:
		p.message = &types.MessageHeaderResponse{}
	case MessageTypeDuplicate:
		var dup types.MessageDuplicate
		p.message = &dup
	case MessageTypeGetMsg:
		p.message = &types.MessageGetMsg{}
	case MessageTypeControl:
		p.message = &types.MessageControl{}
	default:
		return fmt.Errorf("unkown mssage type %v ", p.messageType)
	}
	return nil
}

func (p *p2PMessage) GetMarkHashes() types.Hashes {
	if p.message == nil {
		panic("unmarshal first")
	}
	switch p.messageType {
	case MessageTypeFetchByHashResponse:
		msg := p.message.(*types.MessageSyncResponse)
		return msg.Hashes()
	case MessageTypeNewTxs:
		msg := p.message.(*types.MessageNewTxs)
		return msg.Hashes()
	case MessageTypeTxsResponse:
		msg := p.message.(*types.MessageTxsResponse)
		return msg.Hashes()
	default:
		return nil
	}
	return nil
}

func (m *p2PMessage) Marshal() error {
	if m.marshalState {
		return nil
	}
	if m.message == nil {
		return errors.New("message is nil")
	}
	var err error
	m.data, err = m.message.MarshalMsg(nil)
	if err != nil {
		return err
	}
	m.marshalState = true
	return err
}

func (p *p2PMessage) Unmarshal() error {
	if p.marshalState {
		return nil
	}
	switch p.messageType {
	case MessageTypePing:
		p.message = &types.MessagePing{}
	case MessageTypePong:
		p.message = &types.MessagePong{}
	case MessageTypeFetchByHashRequest:
		p.message = &types.MessageSyncRequest{}
	case MessageTypeFetchByHashResponse:
		p.message = &types.MessageSyncResponse{}
	case MessageTypeNewTx:
		msg := &types.MessageNewTx{}
		_, err := msg.UnmarshalMsg(p.data)
		if err != nil {
			return err
		}
		if msg.RawTx == nil {
			return errors.New("nil content")
		}
		p.message = msg
		p.marshalState = true
		return nil
	case MessageTypeNewSequencer:
		msg := &types.MessageNewSequencer{}
		_, err := msg.UnmarshalMsg(p.data)
		if err != nil {
			return err
		}
		if msg.RawSequencer == nil {
			return errors.New("nil content")
		}
		p.message = msg
		p.marshalState = true
		return nil
	case MessageTypeNewTxs:
		p.message = &types.MessageNewTxs{}
	case MessageTypeSequencerHeader:
		p.message = &types.MessageSequencerHeader{}

	case MessageTypeBodiesRequest:
		p.message = &types.MessageBodiesRequest{}
	case MessageTypeBodiesResponse:
		p.message = &types.MessageBodiesResponse{}

	case MessageTypeTxsRequest:
		p.message = &types.MessageTxsRequest{}
	case MessageTypeTxsResponse:
		p.message = &types.MessageTxsResponse{}
	case MessageTypeHeaderRequest:
		p.message = &types.MessageHeaderRequest{}
	case MessageTypeHeaderResponse:
		p.message = &types.MessageHeaderResponse{}
	case MessageTypeDuplicate:
		var dup types.MessageDuplicate
		p.message = &dup
	case MessageTypeGetMsg:
		msg := &types.MessageGetMsg{}
		_, err := msg.UnmarshalMsg(p.data)
		if err != nil {
			return err
		}
		if msg.Hash == nil {
			return errors.New("nil content")
		}
		p.message = msg
		p.marshalState = true
		return nil
	case MessageTypeControl:
		msg := &types.MessageControl{}
		_, err := msg.UnmarshalMsg(p.data)
		if err != nil {
			return err
		}
		if msg.Hash == nil {
			return errors.New("nil content")
		}
		p.message = msg
		p.marshalState = true
		return nil
	default:
		return fmt.Errorf("unkown mssage type %v ", p.messageType)
	}
	_, err := p.message.UnmarshalMsg(p.data)
	p.marshalState = true
	return err
}

//
func (m *p2PMessage) sendDuplicateMsg() bool {
	return m.messageType == MessageTypeNewTx || m.messageType == MessageTypeNewSequencer
}

type msgKey struct {
	data [types.HashLength + 2]byte
}

func (m *p2PMessage) msgKey() msgKey {
	return newMsgKey(m.messageType, *m.hash)
}

func (k msgKey) GetType() (MessageType, error) {
	if len(k.data) != types.HashLength+2 {
		return 0, errors.New("size err")
	}
	return MessageType(binary.BigEndian.Uint16(k.data[0:2])), nil
}

func (k msgKey) GetHash() (types.Hash, error) {
	if len(k.data) != types.HashLength+2 {
		return types.Hash{}, errors.New("size err")
	}
	return types.BytesToHash(k.data[2:]), nil
}

func newMsgKey(m MessageType, hash types.Hash) msgKey {
	var key msgKey
	b := make([]byte, 2)
	//use one key for tx and sequencer
	if m == MessageTypeNewSequencer {
		m = MessageTypeNewTx
	}
	binary.BigEndian.PutUint16(b, uint16(m))
	copy(key.data[:], b)
	copy(key.data[2:], hash.ToBytes())
	return key
}
