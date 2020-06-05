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

package message_archive

import (
	"fmt"
	types2 "github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/og/types/archive"
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

func (m BinaryMessageType) GetMsg() types.Message {
	var message types.Message
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
		message = &types.MessageCampaign{}
	case MessageTypeTermChange:
		message = &types.MessageTermChange{}
	//case MessageTypeConsensusDkgDeal:
	//	message = &types.MessageConsensusDkgDeal{}
	//case MessageTypeConsensusDkgDealResponse:
	//	message = &types.MessageConsensusDkgDealResponse{}
	//case MessageTypeConsensusDkgSigSets:
	//	message = &types.MessageConsensusDkgSigSets{}
	//case MessageTypeConsensusDkgGenesisPublicKey:
	//	message = &types.MessageConsensusDkgGenesisPublicKey{}

	case MessageTypeTermChangeResponse:
		message = &types.MessageTermChangeResponse{}
	case MessageTypeTermChangeRequest:
		message = &types.MessageTermChangeRequest{}

	case MessageTypeArchive:
		message = &types.MessageNewArchive{}
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
	types.Sequencer
}

func (s *SequencerProposal) String() string {
	return fmt.Sprintf("seqProposal") + s.Sequencer.String()
}

func (s SequencerProposal) Equal(o bft.Proposal) bool {
	v, ok := o.(*SequencerProposal)
	if !ok || v == nil {
		return false
	}
	return s.GetHash() == v.GetHash()
}

func (s SequencerProposal) GetId() *types2.Hash {
	//should copy ?
	var hash types2.Hash
	hash.MustSetBytes(s.GetHash().ToBytes(), types2.PaddingNone)
	return &hash
}

func (s SequencerProposal) Copy() bft.Proposal {
	var r SequencerProposal
	r = s
	return &r
}

