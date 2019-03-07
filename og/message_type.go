// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package og

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types"
	"sync/atomic"
)

//go:generate msgp
//msgp:tuple p2PMessage

const (
	OG01 = 01
	OG02 = 02
)

// ProtocolName is the official short name of the protocol used during capability negotiation.
var ProtocolName = "og"

// ProtocolVersions are the supported versions of the og protocol (first is primary).
var ProtocolVersions = []uint{OG02, OG01}

// ProtocolLengths are the number of implemented message corresponding to different protocol versions.
var ProtocolLengths = []MessageType{MessageTypeOg02Length, MessageTypeOg01Length}

const ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message

//global msg counter , generate global msg request id
var MsgCounter *MessageCounter

type MessageType uint16

// og protocol message codes
const (
	// Protocol messages belonging to OG/01
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

	//for consensus
	MessageTypeCampaign
	MessageTypeTermChange
	MessageTypeConsensusDkgDeal
	MessageTypeConsensusDkgDealResponse
	MessageTypeConsensusDkgSigSets

	MessageTypeSecret

	MessageTypeOg01Length //og01 length

	// Protocol messages belonging to og/02

	GetNodeDataMsg
	NodeDataMsg
	GetReceiptsMsg
	MessageTypeOg02Length
)

type SendingType uint8

const (
	sendingTypeBroacast SendingType = iota
	sendingTypeMulticast
	sendingTypeMulticastToSource
	sendingTypeBroacastWithFilter
	sendingTypeBroacastWithLink
)

func (mt MessageType) isValid() bool {
	if mt >= MessageTypeOg02Length {
		return false
	}
	return true
}

func (mt MessageType) String() string {
	switch mt {
	case StatusMsg:
		return "StatusMsg"
	case MessageTypePing:
		return "MessageTypePing"
	case MessageTypePong:
		return "MessageTypePong"
	case MessageTypeFetchByHashRequest:
		return "MessageTypeFetchByHashRequest"
	case MessageTypeFetchByHashResponse:
		return "MessageTypeFetchByHashResponse"
	case MessageTypeNewTx:
		return "MessageTypeNewTx"
	case MessageTypeNewSequencer:
		return "MessageTypeNewSequencer"
	case MessageTypeNewTxs:
		return "MessageTypeNewTxs"
	case MessageTypeSequencerHeader:
		return "MessageTypeSequencerHeader"

	case MessageTypeBodiesRequest:
		return "MessageTypeBodiesRequest"
	case MessageTypeBodiesResponse:
		return "MessageTypeBodiesResponse"
	case MessageTypeTxsRequest:
		return "MessageTypeTxsRequest"
	case MessageTypeTxsResponse:
		return "MessageTypeTxsResponse"
	case MessageTypeHeaderRequest:
		return "MessageTypeHeaderRequest"
	case MessageTypeHeaderResponse:
		return "MessageTypeHeaderResponse"

		//for optimizing network
	case MessageTypeGetMsg:
		return "MessageTypeGetMsg"
	case MessageTypeDuplicate:
		return "MessageTypeDuplicate"
	case MessageTypeControl:
		return "MessageTypeControl"

		//for consensus
	case MessageTypeCampaign:
		return "MessageTypeCampaign"
	case MessageTypeTermChange:
		return "MessageTypeTermChange"
	case MessageTypeConsensusDkgDeal:
		return "MessageTypeConsensusDkgDeal"
	case MessageTypeConsensusDkgDealResponse:
		return "MessageTypeConsensusDkgDealResponse"

	case MessageTypeConsensusDkgSigSets:
		return "MessageTypeDkgSigSets"

	case MessageTypeSecret:
		return "MessageTypeSecret"

	case MessageTypeOg01Length: //og01 length
		return "MessageTypeOg01Length"

		// Protocol messages belonging to og/02

	case GetNodeDataMsg:
		return "GetNodeDataMsg"
	case NodeDataMsg:
		return "NodeDataMsg"
	case GetReceiptsMsg:
		return "GetReceiptsMsg"
	case MessageTypeOg02Length:
		return "MessageTypeOg02Length"
	default:
		return fmt.Sprintf("unkown message type %d", mt)
	}
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
	encrypt      bool
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
//msgp:tuple StatusData
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
	case MessageTypeCampaign:
		p.message = &types.MessageCampaign{}
	case MessageTypeTermChange:
		p.message = &types.MessageTermChange{}
	case MessageTypeConsensusDkgDeal:
		p.message = &types.MessageConsensusDkgDeal{}
	case MessageTypeConsensusDkgDealResponse:
		p.message = &types.MessageConsensusDkgDealResponse{}

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

func (m *p2PMessage) Encrypt(pub *crypto.PublicKey) error {
	//if m.messageType == MessageTypeConsensusDkgDeal || m.messageType == MessageTypeConsensusDkgDealResponse {
	b := make([]byte, 2)
	//use one key for tx and sequencer
	binary.BigEndian.PutUint16(b, uint16(m.messageType))
	m.data = append(m.data, b[:]...)
	m.encrypt = true
	m.messageType = MessageTypeSecret
	ct, err := pub.Encrypt(m.data)
	if err != nil {
		return err
	}
	m.data = ct
	//add target
	m.data = append(m.data, pub.Bytes[:3]...)
	return nil
}

func (m *p2PMessage) checkRequiredSize() bool {
	if m.messageType == MessageTypeSecret {
		if len(m.data) < 3 {
			return false
		}
	}
	return true
}

func (m *p2PMessage) maybeIsforMe(myPub *crypto.PublicKey) bool {
	if m.messageType != MessageTypeSecret {
		panic("not a secret message")
	}
	//check target
	target := m.data[len(m.data)-3:]
	if !bytes.Equal(target, myPub.Bytes[:3]) {
		//not four me
		return false
	}
	return true
}

func (m *p2PMessage) Decrypt(priv *crypto.PrivateKey) error {
	if m.messageType != MessageTypeSecret {
		panic("not a secret message")
	}
	d := make([]byte, len(m.data)-3)
	copy(d, m.data[:len(m.data)-3])
	msg, err := priv.Decrypt(d)
	if err != nil {
		return err
	}
	if len(msg) < 3 {
		return fmt.Errorf("lengh error %d", len(msg))
	}
	b := make([]byte, 2)
	copy(b, msg[len(msg)-2:])
	mType := binary.BigEndian.Uint16(b)
	m.messageType = MessageType(mType)
	if !m.messageType.isValid() {
		return fmt.Errorf("message type error %s", m.messageType.String())
	}
	m.data = msg[:len(msg)-2]
	return nil
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
	case MessageTypeCampaign:
		p.message = &types.MessageCampaign{}
	case MessageTypeTermChange:
		p.message = &types.MessageTermChange{}
	case MessageTypeConsensusDkgDeal:
		p.message = &types.MessageConsensusDkgDeal{}
	case MessageTypeConsensusDkgDealResponse:
		p.message = &types.MessageConsensusDkgDealResponse{}
	case MessageTypeConsensusDkgSigSets:
		p.message = &types.MessageConsensusDkgSigSets{}
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
