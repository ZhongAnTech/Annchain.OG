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

// ProtocolLengths are the number of implemented message corresponding to different protocol versions.
var ProtocolLengths = []p2p_message.MessageType{p2p_message.MessageTypeOg02Length, p2p_message.MessageTypeOg01Length}

const ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message



type SendingType uint8

const (
	sendingTypeBroacast SendingType = iota
	sendingTypeMulticast
	sendingTypeMulticastToSource
	sendingTypeBroacastWithFilter
	sendingTypeBroacastWithLink
)



type p2PMessage struct {
	messageType    p2p_message.MessageType
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
	case p2p_message.MessageTypeNewTx:
		msg := m.message.(*p2p_message.MessageNewTx)
		hash = msg.GetHash()
		var msgHash common.Hash
		msgHash = *hash
		m.hash = &msgHash
		return
	case p2p_message.MessageTypeControl:
		msg := m.message.(*p2p_message.MessageControl)
		var msgHash common.Hash
		msgHash = *msg.Hash
		m.hash = &msgHash
		return

	case p2p_message.MessageTypeNewSequencer:
		msg := m.message.(*p2p_message.MessageNewSequencer)
		hash = msg.GetHash()
		var msgHash common.Hash
		msgHash = *hash
		m.hash = &msgHash
		return
	case p2p_message.MessageTypeTxsRequest:
		data = append(data, []byte(m.sourceID+"txs")...)
	case p2p_message.MessageTypeBodiesRequest:
		data = append(data, []byte(m.sourceID+"bq")...)
	case p2p_message.MessageTypeTermChangeRequest:
		data = append(data, []byte(m.sourceID+"tq")...)
	case p2p_message.MessageTypeFetchByHashRequest:
		data = append(data, []byte(m.sourceID+"fe")...)
	case p2p_message.MessageTypeHeaderRequest:
		data = append(data, []byte(m.sourceID+"hq")...)
	case p2p_message.MessageTypeHeaderResponse:
		data = append(data, []byte(m.sourceID+"hp")...)
	case p2p_message.MessageTypeBodiesResponse:
		data = append(data, []byte(m.sourceID+"bp")...)
	case p2p_message.MessageTypeSequencerHeader:
		data = append(data, []byte(m.sourceID+"sq")...)
	case p2p_message.MessageTypeGetMsg:
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


func (p *p2PMessage) GetMarkHashes() common.Hashes {
	if p.message == nil {
		panic("unmarshal first")
	}
	switch p.messageType {
	case p2p_message.MessageTypeFetchByHashResponse:
		msg := p.message.(*p2p_message.MessageSyncResponse)
		return msg.Hashes()
	case p2p_message.MessageTypeNewTxs:
		msg := p.message.(*p2p_message.MessageNewTxs)
		return msg.Hashes()
	case p2p_message.MessageTypeTxsResponse:
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
	m.messageType = p2p_message.MessageTypeSecret
	return nil
}

func (m *p2PMessage) Encrypt(pub *crypto.PublicKey) error {
	//if m.messageType == MessageTypeConsensusDkgDeal || m.messageType == MessageTypeConsensusDkgDealResponse {
	b := make([]byte, 2)
	//use one key for tx and sequencer
	binary.BigEndian.PutUint16(b, uint16(m.messageType))
	m.data = append(m.data, b[:]...)
	m.messageType = p2p_message.MessageTypeSecret
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
	if m.messageType == p2p_message.MessageTypeSecret {
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
	if m.messageType != p2p_message.MessageTypeSecret {
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
	m.messageType = p2p_message.MessageType(mType)
	if !m.messageType.IsValid() {
		return fmt.Errorf("message type error %s", m.messageType.String())
	}
	m.data = msg[:len(msg)-2]
	return nil
}

func (m *p2PMessage) Decrypt(priv *crypto.PrivateKey) error {
	if m.messageType != p2p_message.MessageTypeSecret {
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
	m.messageType = p2p_message.MessageType(mType)
	if !m.messageType.IsValid() {
		return fmt.Errorf("message type error %s", m.messageType.String())
	}
	m.data = msg[:len(msg)-2]
	return nil
}

func (p *p2PMessage) Unmarshal() error {
	if p.marshalState {
		return nil
	}
	p.message = p.messageType.GetMsg()
	if p.message ==nil {
		 return fmt.Errorf("unkown mssage type %v ", p.messageType)
	}
	switch p.messageType {

	case p2p_message.MessageTypeNewTx:
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
	case p2p_message.MessageTypeNewSequencer:
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
	case p2p_message.MessageTypeGetMsg:
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
	case p2p_message.MessageTypeControl:
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


	default:
		return fmt.Errorf("unkown mssage type %v ", p.messageType)
	}
	_, err := p.message.UnmarshalMsg(p.data)
	p.marshalState = true
	return err
}

//
func (m *p2PMessage) sendDuplicateMsg() bool {
	return m.messageType == p2p_message.MessageTypeNewTx || m.messageType == p2p_message.MessageTypeNewSequencer
}

func (m *p2PMessage) msgKey() p2p_message.MsgKey {
	return p2p_message.NewMsgKey(m.messageType, *m.hash)
}