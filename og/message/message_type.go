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
	"github.com/annchain/OG/og/protocol/ogmessage"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"
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

func (m BinaryMessageType) GetMsg() ogmessage.Message {
	var message ogmessage.Message
	switch m {
	case MessageTypeNewTx:
		message = &archive.MessageNewTx{}
	case MessageTypeNewSequencer:
		message = &archive.MessageNewSequencer{}
	case MessageTypeGetMsg:
		message = &archive.MessageGetMsg{}
	case MessageTypeControl:
		message = &archive.MessageControl{}
	case MessageTypeCampaign:
		message = &ogmessage.MessageCampaign{}
	case MessageTypeTermChange:
		message = &ogmessage.MessageTermChange{}
	//case MessageTypeConsensusDkgDeal:
	//	message = &ogmessage.MessageConsensusDkgDeal{}
	//case MessageTypeConsensusDkgDealResponse:
	//	message = &ogmessage.MessageConsensusDkgDealResponse{}
	//case MessageTypeConsensusDkgSigSets:
	//	message = &ogmessage.MessageConsensusDkgSigSets{}
	//case MessageTypeConsensusDkgGenesisPublicKey:
	//	message = &ogmessage.MessageConsensusDkgGenesisPublicKey{}

	case MessageTypeTermChangeResponse:
		message = &ogmessage.MessageTermChangeResponse{}
	case MessageTypeTermChangeRequest:
		message = &ogmessage.MessageTermChangeRequest{}

	case MessageTypeArchive:
		message = &ogmessage.MessageNewArchive{}
	case MessageTypeActionTX:
		message = &archive.MessageNewActionTx{}

	//case MessageTypeProposal:
	//	message = &bft.BftMessageProposal{
	//		Value: &bft.SequencerProposal{},
	//	}
	//case MessageTypePreVote:
	//	message = &bft.BftMessagePreVote{}
	//case MessageTypePreCommit:
	//	message = &bft.BftMessagePreCommit{}
	case MessageTypeNewTxs:
		message = &archive.MessageNewTxs{}
	case MessageTypeSequencerHeader:
		message = &archive.MessageSequencerHeader{}

	case MessageTypeBodiesRequest:
		message = &archive.MessageBodiesRequest{}
	case MessageTypeBodiesResponse:
		message = &archive.MessageBodiesResponse{}

	case MessageTypeTxsRequest:
		message = &archive.MessageTxsRequest{}
	case MessageTypeTxsResponse:
		message = &archive.MessageTxsResponse{}
	case MessageTypeHeaderRequest:
		message = &archive.MessageHeaderRequest{}
	case MessageTypeHeaderResponse:
		message = &archive.MessageHeaderResponse{}
	case MessageTypeDuplicate:
		var dup archive.MessageDuplicate
		message = &dup
	case MessageTypePing:
		message = &archive.MessagePing{}
	case MessageTypePong:
		message = &archive.MessagePong{}
	case MessageTypeFetchByHashRequest:
		message = &archive.MessageSyncRequest{}
	case MessageTypeFetchByHashResponse:
		message = &archive.MessageSyncResponse{}
	default:
		return nil
	}
	return message
}


//msgp:tuple SequencerProposal
type SequencerProposal struct {
	ogmessage.Sequencer
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

