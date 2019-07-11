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
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types/p2p_message"
	"sync/atomic"
)

//go:generate msgp

const (
	OG01 = 01
	OG02 = 02
)

// ProtocolName is the official short name of the protocol used during capability negotiation.
var ProtocolName = "og"

// ProtocolVersions are the supported versions of the og protocol (first is primary).
var ProtocolVersions = []uint32{OG02, OG01}

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

	MessageTypeArchive
	MessageTypeActionTX

	//for consensus dkg
	MessageTypeConsensusDkgDeal
	MessageTypeConsensusDkgDealResponse
	MessageTypeConsensusDkgSigSets

	MessageTypeConsensusDkgGenesisPublicKey

	MessageTypeTermChangeRequest
	MessageTypeTermChangeResponse

	MessageTypeSecret //encrypted message

	MessageTypeProposal
	MessageTypePreVote
	MessageTypePreCommit

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
	case MessageTypeArchive:
		return "MessageTypeArchive"
	case MessageTypeActionTX:
		return "MessageTypeActionTX"

	case MessageTypeConsensusDkgDeal:
		return "MessageTypeConsensusDkgDeal"
	case MessageTypeConsensusDkgDealResponse:
		return "MessageTypeConsensusDkgDealResponse"

	case MessageTypeConsensusDkgSigSets:
		return "MessageTypeDkgSigSets"

	case MessageTypeConsensusDkgGenesisPublicKey:
		return "MessageTypeConsensusDkgGenesisPublicKey"
	case MessageTypeTermChangeRequest:
		return "MessageTypeTermChangeRequest"
	case MessageTypeTermChangeResponse:
		return "MessageTypeTermChangeResponse"
	case MessageTypeSecret:
		return "MessageTypeSecret"

	case MessageTypeProposal:
		return "MessageTypeProposal"
	case MessageTypePreVote:
		return "MessageTypePreVote"
	case MessageTypePreCommit:
		return "MessageTypePreCommit"

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
	messageType    MessageType
	data           []byte
	hash           *common.Hash //inner use to avoid resend a message to the same peer
	sourceID       string       // the source that this message  coming from , outgoing if it is nil
	sendingType    SendingType  //sending type
	version        int          // peer version.
	message        p2p_message.Message
	sourceHash     *common.Hash
	marshalState   bool
	disableEncrypt bool
}

func (m *p2PMessage) calculateHash() {
	// for txs,or response msg , even if  source peer id is different ,they were duplicated txs
	//for request ,if source id is different they were different  msg ,don't drop it
	//if we dropped header response because of duplicate , header request will time out
	if len(m.data) == 0 {
		msgLog.Error("nil data to calculate hash")
	}
	data := m.data
	var hash *common.Hash
	switch m.messageType {
	case MessageTypeNewTx:
		msg := m.message.(*p2p_message.MessageNewTx)
		hash = msg.GetHash()
		var msgHash common.Hash
		msgHash = *hash
		m.hash = &msgHash
		return
	case MessageTypeControl:
		msg := m.message.(*p2p_message.MessageControl)
		var msgHash common.Hash
		msgHash = *msg.Hash
		m.hash = &msgHash
		return

	case MessageTypeNewSequencer:
		msg := m.message.(*p2p_message.MessageNewSequencer)
		hash = msg.GetHash()
		var msgHash common.Hash
		msgHash = *hash
		m.hash = &msgHash
		return
	case MessageTypeTxsRequest:
		data = append(data, []byte(m.sourceID+"txs")...)
	case MessageTypeBodiesRequest:
		data = append(data, []byte(m.sourceID+"bq")...)
	case MessageTypeTermChangeRequest:
		data = append(data, []byte(m.sourceID+"tq")...)
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
	m.hash = &common.Hash{}
	m.hash.MustSetBytes(sum, common.PaddingNone)
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
	CurrentBlock    common.Hash
	GenesisBlock    common.Hash
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

func (p *p2PMessage) GetMarkHashes() common.Hashes {
	if p.message == nil {
		panic("unmarshal first")
	}
	switch p.messageType {
	case MessageTypeFetchByHashResponse:
		msg := p.message.(*p2p_message.MessageSyncResponse)
		return msg.Hashes()
	case MessageTypeNewTxs:
		msg := p.message.(*p2p_message.MessageNewTxs)
		return msg.Hashes()
	case MessageTypeTxsResponse:
		msg := p.message.(*p2p_message.MessageTxsResponse)
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

func (m *p2PMessage) appendGossipTarget(pub *crypto.PublicKey) error {
	b := make([]byte, 2)
	//use one key for tx and sequencer
	binary.BigEndian.PutUint16(b, uint16(m.messageType))
	m.data = append(m.data, b[:]...)
	m.disableEncrypt = true
	m.data = append(m.data, pub.Bytes[:8]...)
	m.messageType = MessageTypeSecret
	return nil
}

func (m *p2PMessage) Encrypt(pub *crypto.PublicKey) error {
	//if m.messageType == MessageTypeConsensusDkgDeal || m.messageType == MessageTypeConsensusDkgDealResponse {
	b := make([]byte, 2)
	//use one key for tx and sequencer
	binary.BigEndian.PutUint16(b, uint16(m.messageType))
	m.data = append(m.data, b[:]...)
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
		if m.disableEncrypt {
			if len(m.data) < 8 {
				return false
			}
		}
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
	if m.disableEncrypt {
		target := m.data[len(m.data)-8:]
		if !bytes.Equal(target, myPub.Bytes[:8]) {
			//not four me
			return false
		}
		return true
	}
	target := m.data[len(m.data)-3:]
	if !bytes.Equal(target, myPub.Bytes[:3]) {
		//not four me
		return false
	}
	return true
}

func (m *p2PMessage) removeGossipTarget() error {
	msg := make([]byte, len(m.data)-8)
	copy(msg, m.data[:len(m.data)-8])
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
		p.message = &p2p_message.MessagePing{}
	case MessageTypePong:
		p.message = &p2p_message.MessagePong{}
	case MessageTypeFetchByHashRequest:
		p.message = &p2p_message.MessageSyncRequest{}
	case MessageTypeFetchByHashResponse:
		p.message = &p2p_message.MessageSyncResponse{}
	case MessageTypeNewTx:
		msg := &p2p_message.MessageNewTx{}
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
		msg := &p2p_message.MessageNewSequencer{}
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
		p.message = &p2p_message.MessageNewTxs{}
	case MessageTypeSequencerHeader:
		p.message = &p2p_message.MessageSequencerHeader{}

	case MessageTypeBodiesRequest:
		p.message = &p2p_message.MessageBodiesRequest{}
	case MessageTypeBodiesResponse:
		p.message = &p2p_message.MessageBodiesResponse{}

	case MessageTypeTxsRequest:
		p.message = &p2p_message.MessageTxsRequest{}
	case MessageTypeTxsResponse:
		p.message = &p2p_message.MessageTxsResponse{}
	case MessageTypeHeaderRequest:
		p.message = &p2p_message.MessageHeaderRequest{}
	case MessageTypeHeaderResponse:
		p.message = &p2p_message.MessageHeaderResponse{}
	case MessageTypeDuplicate:
		var dup p2p_message.MessageDuplicate
		p.message = &dup
	case MessageTypeGetMsg:
		msg := &p2p_message.MessageGetMsg{}
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
		msg := &p2p_message.MessageControl{}
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
		p.message = &p2p_message.MessageCampaign{}
	case MessageTypeTermChange:
		p.message = &p2p_message.MessageTermChange{}
	case MessageTypeConsensusDkgDeal:
		p.message = &p2p_message.MessageConsensusDkgDeal{}
	case MessageTypeConsensusDkgDealResponse:
		p.message = &p2p_message.MessageConsensusDkgDealResponse{}
	case MessageTypeConsensusDkgSigSets:
		p.message = &p2p_message.MessageConsensusDkgSigSets{}
	case MessageTypeConsensusDkgGenesisPublicKey:
		p.message = &p2p_message.MessageConsensusDkgGenesisPublicKey{}

	case MessageTypeTermChangeResponse:
		p.message = &p2p_message.MessageTermChangeResponse{}
	case MessageTypeTermChangeRequest:
		p.message = &p2p_message.MessageTermChangeRequest{}

	case MessageTypeArchive:
		p.message = &p2p_message.MessageNewArchive{}
	case MessageTypeActionTX:
		p.message = &p2p_message.MessageNewActionTx{}

	case MessageTypeProposal:
		p.message = &p2p_message.MessageProposal{
			Value: &p2p_message.SequencerProposal{},
		}
	case MessageTypePreVote:
		p.message = &p2p_message.MessagePreVote{}
	case MessageTypePreCommit:
		p.message = &p2p_message.MessagePreCommit{}

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
	data [common.HashLength + 2]byte
}

func (m *p2PMessage) msgKey() msgKey {
	return newMsgKey(m.messageType, *m.hash)
}

func (k msgKey) GetType() (MessageType, error) {
	if len(k.data) != common.HashLength+2 {
		return 0, errors.New("size err")
	}
	return MessageType(binary.BigEndian.Uint16(k.data[0:2])), nil
}

func (k msgKey) GetHash() (common.Hash, error) {
	if len(k.data) != common.HashLength+2 {
		return common.Hash{}, errors.New("size err")
	}
	return common.BytesToHash(k.data[2:]), nil
}

func newMsgKey(m MessageType, hash common.Hash) msgKey {
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
