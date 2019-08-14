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
	"github.com/annchain/OG/og/message"
	"github.com/annchain/OG/types/p2p_message"
	"sync/atomic"
)

////go:generate msgp

const (
	OG01 = 01
	OG02 = 02
)

// ProtocolName is the official short name of the protocol used during capability negotiation.
var ProtocolName = "og"

// ProtocolVersions are the supported versions of the og protocol (first is primary).
var ProtocolVersions = []uint32{OG02, OG01}

// ProtocolLengths are the number of implemented Message corresponding to different protocol versions.
var ProtocolLengths = []message.OGMessageType{message.MessageTypeOg02Length, message.MessageTypeOg01Length}

const ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol Message

type SendingType uint8

const (
	sendingTypeBroadcast SendingType = iota
	sendingTypeMulticast
	sendingTypeMulticastToSource
	sendingTypeBroacastWithFilter
	sendingTypeBroacastWithLink
)

type OGMessage struct {
	MessageType    message.OGMessageType
	Data           []byte
	Hash           *common.Hash //inner use to avoid resend a Message to the same peer
	SourceID       string       // the source that this Message  coming from , outgoing if it is nil
	SendingType    SendingType  //sending type
	Version        int          // peer Version.
	Message        p2p_message.Message
	SourceHash     *common.Hash
	MarshalState   bool
	DisableEncrypt bool
}

func (m *OGMessage) calculateHash() {
	// for txs,or response msg , even if  source peer id is different ,they were duplicated txs
	//for request ,if source id is different they were different  msg ,don't drop it
	//if we dropped header response because of duplicate , header request will time out
	if len(m.Data) == 0 {
		msgLog.Error("nil Data to calculate Hash")
	}
	data := m.Data
	var hash *common.Hash
	switch m.MessageType {
	case message.MessageTypeNewTx:
		msg := m.Message.(*p2p_message.MessageNewTx)
		hash = msg.GetHash()
		var msgHash common.Hash
		msgHash = *hash
		m.Hash = &msgHash
		return
	case message.MessageTypeControl:
		msg := m.Message.(*p2p_message.MessageControl)
		var msgHash common.Hash
		msgHash = *msg.Hash
		m.Hash = &msgHash
		return

	case message.MessageTypeNewSequencer:
		msg := m.Message.(*p2p_message.MessageNewSequencer)
		hash = msg.GetHash()
		var msgHash common.Hash
		msgHash = *hash
		m.Hash = &msgHash
		return
	case message.MessageTypeTxsRequest:
		data = append(data, []byte(m.SourceID+"txs")...)
	case message.MessageTypeBodiesRequest:
		data = append(data, []byte(m.SourceID+"bq")...)
	case message.MessageTypeTermChangeRequest:
		data = append(data, []byte(m.SourceID+"tq")...)
	case message.MessageTypeFetchByHashRequest:
		data = append(data, []byte(m.SourceID+"fe")...)
	case message.MessageTypeHeaderRequest:
		data = append(data, []byte(m.SourceID+"hq")...)
	case message.MessageTypeHeaderResponse:
		data = append(data, []byte(m.SourceID+"hp")...)
	case message.MessageTypeBodiesResponse:
		data = append(data, []byte(m.SourceID+"bp")...)
	case message.MessageTypeSequencerHeader:
		data = append(data, []byte(m.SourceID+"sq")...)
	case message.MessageTypeGetMsg:
		data = append(data, []byte(m.SourceID+"gm")...)
	default:
	}
	h := sha256.New()
	h.Write(data)
	sum := h.Sum(nil)
	m.Hash = &common.Hash{}
	m.Hash.MustSetBytes(sum, common.PaddingNone)
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
	ErrDecode:                  "Invalid Message",
	ErrInvalidMsgCode:          "Invalid Message code",
	ErrProtocolVersionMismatch: "Protocol Version mismatch",
	ErrNetworkIdMismatch:       "NetworkId mismatch",
	ErrGenesisBlockMismatch:    "Genesis block mismatch",
	ErrNoStatusMsg:             "No status Message",
	ErrExtraStatusMsg:          "Extra status Message",
	ErrSuspendedPeer:           "Suspended peer",
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

func (p *OGMessage) GetMarkHashes() common.Hashes {
	if p.Message == nil {
		panic("unmarshal first")
	}
	switch p.MessageType {
	case message.MessageTypeFetchByHashResponse:
		msg := p.Message.(*p2p_message.MessageSyncResponse)
		return msg.Hashes()
	case message.MessageTypeNewTxs:
		msg := p.Message.(*p2p_message.MessageNewTxs)
		return msg.Hashes()
	case message.MessageTypeTxsResponse:
		msg := p.Message.(*p2p_message.MessageTxsResponse)
		return msg.Hashes()
	default:
		return nil
	}
	return nil
}

func (m *OGMessage) Marshal() error {
	if m.MarshalState {
		return nil
	}
	if m.Message == nil {
		return errors.New("Message is nil")
	}
	var err error
	m.Data, err = m.Message.MarshalMsg(nil)
	if err != nil {
		return err
	}
	m.MarshalState = true
	return err
}

func (m *OGMessage) appendGossipTarget(pub *crypto.PublicKey) error {
	b := make([]byte, 2)
	//use one key for tx and sequencer
	binary.BigEndian.PutUint16(b, uint16(m.MessageType))
	m.Data = append(m.Data, b[:]...)
	m.DisableEncrypt = true
	m.Data = append(m.Data, pub.Bytes[:8]...)
	m.MessageType = message.MessageTypeSecret
	return nil
}

func (m *OGMessage) Encrypt(pub *crypto.PublicKey) error {
	//if m.MessageType == MessageTypeConsensusDkgDeal || m.MessageType == MessageTypeConsensusDkgDealResponse {
	b := make([]byte, 2)
	//use one key for tx and sequencer
	binary.BigEndian.PutUint16(b, uint16(m.MessageType))
	m.Data = append(m.Data, b[:]...)
	m.MessageType = message.MessageTypeSecret
	ct, err := pub.Encrypt(m.Data)
	if err != nil {
		return err
	}
	m.Data = ct
	//add target
	m.Data = append(m.Data, pub.Bytes[:3]...)
	return nil
}

func (m *OGMessage) checkRequiredSize() bool {
	if m.MessageType == message.MessageTypeSecret {
		if m.DisableEncrypt {
			if len(m.Data) < 8 {
				return false
			}
		}
		if len(m.Data) < 3 {
			return false
		}
	}
	return true
}

func (m *OGMessage) maybeIsforMe(myPub *crypto.PublicKey) bool {
	if m.MessageType != message.MessageTypeSecret {
		panic("not a secret Message")
	}
	//check target
	if m.DisableEncrypt {
		target := m.Data[len(m.Data)-8:]
		if !bytes.Equal(target, myPub.Bytes[:8]) {
			//not four me
			return false
		}
		return true
	}
	target := m.Data[len(m.Data)-3:]
	if !bytes.Equal(target, myPub.Bytes[:3]) {
		//not four me
		return false
	}
	return true
}

func (m *OGMessage) removeGossipTarget() error {
	msg := make([]byte, len(m.Data)-8)
	copy(msg, m.Data[:len(m.Data)-8])
	if len(msg) < 3 {
		return fmt.Errorf("lengh error %d", len(msg))
	}
	b := make([]byte, 2)
	copy(b, msg[len(msg)-2:])
	mType := binary.BigEndian.Uint16(b)
	m.MessageType = message.OGMessageType(mType)
	if !m.MessageType.IsValid() {
		return fmt.Errorf("Message type error %s", m.MessageType.String())
	}
	m.Data = msg[:len(msg)-2]
	return nil
}

func (m *OGMessage) Decrypt(priv *crypto.PrivateKey) error {
	if m.MessageType != message.MessageTypeSecret {
		panic("not a secret Message")
	}
	d := make([]byte, len(m.Data)-3)
	copy(d, m.Data[:len(m.Data)-3])
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
	m.MessageType = message.OGMessageType(mType)
	if !m.MessageType.IsValid() {
		return fmt.Errorf("Message type error %s", m.MessageType.String())
	}
	m.Data = msg[:len(msg)-2]
	return nil
}

func (p *OGMessage) Unmarshal() error {
	if p.MarshalState {
		return nil
	}
	p.Message = p.MessageType.GetMsg()
	if p.Message == nil {
		return fmt.Errorf("unknown Message type %v ", p.MessageType)
	}
	switch p.MessageType {

	case message.MessageTypeNewTx:
		msg := &p2p_message.MessageNewTx{}
		_, err := msg.UnmarshalMsg(p.Data)
		if err != nil {
			return err
		}
		if msg.RawTx == nil {
			return errors.New("nil content")
		}
		p.Message = msg
		p.MarshalState = true
		return nil
	case message.MessageTypeNewSequencer:
		msg := &p2p_message.MessageNewSequencer{}
		_, err := msg.UnmarshalMsg(p.Data)
		if err != nil {
			return err
		}
		if msg.RawSequencer == nil {
			return errors.New("nil content")
		}
		p.Message = msg
		p.MarshalState = true
		return nil
	case message.MessageTypeGetMsg:
		msg := &p2p_message.MessageGetMsg{}
		_, err := msg.UnmarshalMsg(p.Data)
		if err != nil {
			return err
		}
		if msg.Hash == nil {
			return errors.New("nil content")
		}
		p.Message = msg
		p.MarshalState = true
		return nil
	case message.MessageTypeControl:
		msg := &p2p_message.MessageControl{}
		_, err := msg.UnmarshalMsg(p.Data)
		if err != nil {
			return err
		}
		if msg.Hash == nil {
			return errors.New("nil content")
		}
		p.Message = msg
		p.MarshalState = true
		return nil
	default:
	}
	_, err := p.Message.UnmarshalMsg(p.Data)
	p.MarshalState = true
	return err
}

//
func (m *OGMessage) sendDuplicateMsg() bool {
	return m.MessageType == message.MessageTypeNewTx || m.MessageType == message.MessageTypeNewSequencer
}

func (m *OGMessage) msgKey() message.MsgKey {
	return message.NewMsgKey(m.MessageType, *m.Hash)
}
