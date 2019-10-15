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
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/protocol_message"
)

//go:generate msgp

//global msg counter , generate global msg request id
var MsgCounter *MessageCounter

type SendingType uint8

const (
	SendingTypeBroadcast SendingType = iota
	SendingTypeMulticast
	SendingTypeMulticastToSource
	SendingTypeBroacastWithFilter
	SendingTypeBroacastWithLink
)

//func (mt BinaryMessageType) IsValid() bool {
//	if mt >= MessageTypeOg02Length {
//		return false
//	}
//	return true
//}


func (m BinaryMessageType) GetMsg() p2p_message.Message {
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


//msgp:tuple SequencerProposal
type SequencerProposal struct {
	protocol_message.Sequencer
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

