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
		message = &ogmessage.MessageNewTx{}
	case MessageTypeNewSequencer:
		message = &ogmessage.MessageNewSequencer{}
	case MessageTypeGetMsg:
		message = &ogmessage.MessageGetMsg{}
	case MessageTypeControl:
		message = &ogmessage.MessageControl{}
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
		message = &ogmessage.MessageNewActionTx{}

	//case MessageTypeProposal:
	//	message = &bft.BftMessageProposal{
	//		Value: &bft.SequencerProposal{},
	//	}
	//case MessageTypePreVote:
	//	message = &bft.BftMessagePreVote{}
	//case MessageTypePreCommit:
	//	message = &bft.BftMessagePreCommit{}
	case MessageTypeNewTxs:
		message = &ogmessage.MessageNewTxs{}
	case MessageTypeSequencerHeader:
		message = &ogmessage.MessageSequencerHeader{}

	case MessageTypeBodiesRequest:
		message = &ogmessage.MessageBodiesRequest{}
	case MessageTypeBodiesResponse:
		message = &ogmessage.MessageBodiesResponse{}

	case MessageTypeTxsRequest:
		message = &ogmessage.MessageTxsRequest{}
	case MessageTypeTxsResponse:
		message = &ogmessage.MessageTxsResponse{}
	case MessageTypeHeaderRequest:
		message = &ogmessage.MessageHeaderRequest{}
	case MessageTypeHeaderResponse:
		message = &ogmessage.MessageHeaderResponse{}
	case MessageTypeDuplicate:
		var dup ogmessage.MessageDuplicate
		message = &dup
	case MessageTypePing:
		message = &ogmessage.MessagePing{}
	case MessageTypePong:
		message = &ogmessage.MessagePong{}
	case MessageTypeFetchByHashRequest:
		message = &ogmessage.MessageSyncRequest{}
	case MessageTypeFetchByHashResponse:
		message = &ogmessage.MessageSyncResponse{}
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

