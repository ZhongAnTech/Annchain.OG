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

package message

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/annchain/OG/types/tx_types"
	"sync/atomic"
)

//go:generate msgp

//global msg counter , generate global msg request id
var MsgCounter *MessageCounter

type OGMessageType uint16

// og protocol message codes
// TODO: use MessageTypeManager to manage global messages
// basic messages ids range from [0, 100)
// bft consensus: [100, 200)
// dkg: [200, 300)
const (
	// Protocol messages belonging to OG/01
	StatusMsg OGMessageType = iota + 0
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

	//move to dkg package
	//MessageTypeConsensusDkgDeal
	//MessageTypeConsensusDkgDealResponse
	//MessageTypeConsensusDkgSigSets
	//MessageTypeConsensusDkgGenesisPublicKey

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

const (
	SendingTypeBroadcast SendingType = iota
	SendingTypeMulticast
	SendingTypeMulticastToSource
	SendingTypeBroacastWithFilter
	SendingTypeBroacastWithLink
)

func (mt OGMessageType) IsValid() bool {
	if mt >= MessageTypeOg02Length {
		return false
	}
	return true
}

func (mt OGMessageType) String() string {
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

	//case MessageTypeConsensusDkgDeal:
	//	return "MessageTypeConsensusDkgDeal"
	//case MessageTypeConsensusDkgDealResponse:
	//	return "MessageTypeConsensusDkgDealResponse"
	//case MessageTypeConsensusDkgSigSets:
	//	return "MessageTypeDkgSigSets"
	//case MessageTypeConsensusDkgGenesisPublicKey:
	//	return "MessageTypeConsensusDkgGenesisPublicKey"
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

func (mt OGMessageType) Code() p2p.MsgCodeType {
	return p2p.MsgCodeType(mt)
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

func (k MsgKey) GetType() (OGMessageType, error) {
	if len(k.data) != common.HashLength+2 {
		return 0, errors.New("size err")
	}
	return OGMessageType(binary.BigEndian.Uint16(k.data[0:2])), nil
}

func (k MsgKey) GetHash() (common.Hash, error) {
	if len(k.data) != common.HashLength+2 {
		return common.Hash{}, errors.New("size err")
	}
	return common.BytesToHash(k.data[2:]), nil
}

func NewMsgKey(m OGMessageType, hash common.Hash) MsgKey {
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

func (m OGMessageType) GetMsg() p2p_message.Message {
	var message p2p_message.Message
	switch m {
	case MessageTypeNewTx:
		message = &p2p_message.MessageNewTx{}
	case MessageTypeNewSequencer:
		message = &p2p_message.MessageNewSequencer{}
	case MessageTypeGetMsg:
		message = &p2p_message.MessageGetMsg{}
	case MessageTypeControl:
		message = &p2p_message.MessageControl{}
	case MessageTypeCampaign:
		message = &p2p_message.MessageCampaign{}
	case MessageTypeTermChange:
		message = &p2p_message.MessageTermChange{}
	//case MessageTypeConsensusDkgDeal:
	//	message = &p2p_message.MessageConsensusDkgDeal{}
	//case MessageTypeConsensusDkgDealResponse:
	//	message = &p2p_message.MessageConsensusDkgDealResponse{}
	//case MessageTypeConsensusDkgSigSets:
	//	message = &p2p_message.MessageConsensusDkgSigSets{}
	//case MessageTypeConsensusDkgGenesisPublicKey:
	//	message = &p2p_message.MessageConsensusDkgGenesisPublicKey{}

	case MessageTypeTermChangeResponse:
		message = &p2p_message.MessageTermChangeResponse{}
	case MessageTypeTermChangeRequest:
		message = &p2p_message.MessageTermChangeRequest{}

	case MessageTypeArchive:
		message = &p2p_message.MessageNewArchive{}
	case MessageTypeActionTX:
		message = &p2p_message.MessageNewActionTx{}

	//case MessageTypeProposal:
	//	message = &bft.MessageProposal{
	//		Value: &bft.SequencerProposal{},
	//	}
	//case MessageTypePreVote:
	//	message = &bft.MessagePreVote{}
	//case MessageTypePreCommit:
	//	message = &bft.MessagePreCommit{}
	case MessageTypeNewTxs:
		message = &p2p_message.MessageNewTxs{}
	case MessageTypeSequencerHeader:
		message = &p2p_message.MessageSequencerHeader{}

	case MessageTypeBodiesRequest:
		message = &p2p_message.MessageBodiesRequest{}
	case MessageTypeBodiesResponse:
		message = &p2p_message.MessageBodiesResponse{}

	case MessageTypeTxsRequest:
		message = &p2p_message.MessageTxsRequest{}
	case MessageTypeTxsResponse:
		message = &p2p_message.MessageTxsResponse{}
	case MessageTypeHeaderRequest:
		message = &p2p_message.MessageHeaderRequest{}
	case MessageTypeHeaderResponse:
		message = &p2p_message.MessageHeaderResponse{}
	case MessageTypeDuplicate:
		var dup p2p_message.MessageDuplicate
		message = &dup
	case MessageTypePing:
		message = &p2p_message.MessagePing{}
	case MessageTypePong:
		message = &p2p_message.MessagePong{}
	case MessageTypeFetchByHashRequest:
		message = &p2p_message.MessageSyncRequest{}
	case MessageTypeFetchByHashResponse:
		message = &p2p_message.MessageSyncResponse{}
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


// SignedOgPartnerMessage is the message that is signed by partner.
// Consensus layer does not need to care about the signing. It is TrustfulPartnerCommunicator's job
//msgp:tuple SignedOgPartnerMessage
type SignedOgPartnerMessage struct {
	bft.BftMessage
	TermId     uint32
	//ValidRound int
	//PublicKey  []byte
	Signature hexutil.Bytes
	PublicKey hexutil.Bytes
}


//msgp:tuple SequencerProposal
type SequencerProposal struct {
	tx_types.Sequencer
}

func (s *SequencerProposal) String() string {
	return fmt.Sprintf("seqProposal") + s.Sequencer.String()
}

func (s SequencerProposal) Equal(o bft.Proposal) bool {
	v, ok := o.(*SequencerProposal)
	if !ok || v == nil {
		return false
	}
	return s.GetTxHash() == v.GetTxHash()
}

func (s SequencerProposal) GetId() *common.Hash {
	//should copy ?
	var hash common.Hash
	hash.MustSetBytes(s.GetTxHash().ToBytes(), common.PaddingNone)
	return &hash
}

func (s SequencerProposal) Copy() bft.Proposal {
	var r SequencerProposal
	r = s
	return &r
}

