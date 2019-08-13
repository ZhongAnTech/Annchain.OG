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

package p2p_message

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/p2p"
	"sync/atomic"
)

//go:generate msgp

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
	MessageTypeConsensus

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

	// move to bft package
	//MessageTypeProposal
	//MessageTypePreVote
	//MessageTypePreCommit

	MessageTypeOg01Length //og01 length

	// Protocol messages belonging to og/02

	GetNodeDataMsg
	NodeDataMsg
	GetReceiptsMsg
	MessageTypeOg02Length
)

type SendingType uint8

func (mt MessageType) IsValid() bool {
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
	case MessageTypeConsensus:
		return "MessageTypeConsensus"
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

	//case MessageTypeProposal:
	//	return "MessageTypeProposal"
	//case MessageTypePreVote:
	//	return "MessageTypePreVote"
	//case MessageTypePreCommit:
	//	return "MessageTypePreCommit"

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

func MsgCountInit() {
	MsgCounter = &MessageCounter{
		requestId: 1,
	}
}

type MsgKey struct {
	data [common.HashLength + 2]byte
}

func (k MsgKey) GetType() (MessageType, error) {
	if len(k.data) != common.HashLength+2 {
		return 0, errors.New("size err")
	}
	return MessageType(binary.BigEndian.Uint16(k.data[0:2])), nil
}

func (k MsgKey) GetHash() (common.Hash, error) {
	if len(k.data) != common.HashLength+2 {
		return common.Hash{}, errors.New("size err")
	}
	return common.BytesToHash(k.data[2:]), nil
}

func NewMsgKey(m MessageType, hash common.Hash) MsgKey {
	var key MsgKey
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

func (m MessageType) GetMsg() Message {
	var message Message
	switch m {
	case MessageTypeNewTx:
		message = &MessageNewTx{}
	case MessageTypeNewSequencer:
		message = &MessageNewSequencer{}
	case MessageTypeGetMsg:
		message = &MessageGetMsg{}
	case MessageTypeControl:
		message = &MessageControl{}
	case MessageTypeCampaign:
		message = &MessageCampaign{}
	case MessageTypeTermChange:
		message = &MessageTermChange{}
	case MessageTypeConsensusDkgDeal:
		message = &MessageConsensusDkgDeal{}
	case MessageTypeConsensusDkgDealResponse:
		message = &MessageConsensusDkgDealResponse{}
	case MessageTypeConsensusDkgSigSets:
		message = &MessageConsensusDkgSigSets{}
	case MessageTypeConsensusDkgGenesisPublicKey:
		message = &MessageConsensusDkgGenesisPublicKey{}

	case MessageTypeTermChangeResponse:
		message = &MessageTermChangeResponse{}
	case MessageTypeTermChangeRequest:
		message = &MessageTermChangeRequest{}

	case MessageTypeArchive:
		message = &MessageNewArchive{}
	case MessageTypeActionTX:
		message = &MessageNewActionTx{}

	case MessageTypeProposal:
		message = &bft.MessageProposal{
			Value: &bft.SequencerProposal{},
		}
	case MessageTypePreVote:
		message = &bft.MessagePreVote{}
	case MessageTypePreCommit:
		message = &bft.MessagePreCommit{}
	case MessageTypeNewTxs:
		message = &MessageNewTxs{}
	case MessageTypeSequencerHeader:
		message = &MessageSequencerHeader{}

	case MessageTypeBodiesRequest:
		message = &MessageBodiesRequest{}
	case MessageTypeBodiesResponse:
		message = &MessageBodiesResponse{}

	case MessageTypeTxsRequest:
		message = &MessageTxsRequest{}
	case MessageTypeTxsResponse:
		message = &MessageTxsResponse{}
	case MessageTypeHeaderRequest:
		message = &MessageHeaderRequest{}
	case MessageTypeHeaderResponse:
		message = &MessageHeaderResponse{}
	case MessageTypeDuplicate:
		var dup MessageDuplicate
		message = &dup
	case MessageTypePing:
		message = &MessagePing{}
	case MessageTypePong:
		message = &MessagePong{}
	case MessageTypeFetchByHashRequest:
		message = &MessageSyncRequest{}
	case MessageTypeFetchByHashResponse:
		message = &MessageSyncResponse{}
	default:
		return nil
	}
	return message
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
