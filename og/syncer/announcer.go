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
package syncer

import (
	"github.com/annchain/OG/og/message"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/annchain/OG/types/tx_types"
)

type Announcer struct {
	messageSender MessageSender
}

func NewAnnouncer(messageSender MessageSender) *Announcer {
	return &Announcer{
		messageSender: messageSender,
	}
}

//BroadcastNewTx brodcast newly created txi message
func (m *Announcer) BroadcastNewTx(txi types.Txi) {
	switch tx := txi.(type) {
	case *tx_types.Tx:
		msgTx := p2p_message.MessageNewTx{RawTx: tx.RawTx()}
		m.messageSender.BroadcastMessageWithLink(message.MessageTypeNewTx, &msgTx)
	case *tx_types.Sequencer:
		msgTx := p2p_message.MessageNewSequencer{RawSequencer: tx.RawSequencer()}
		m.messageSender.BroadcastMessageWithLink(message.MessageTypeNewSequencer, &msgTx)
	case *tx_types.Campaign:
		msg := p2p_message.MessageCampaign{
			RawCampaign: tx.RawCampaign(),
		}
		m.messageSender.BroadcastMessage(message.MessageTypeCampaign, &msg)
	case *tx_types.TermChange:
		msg := p2p_message.MessageTermChange{
			RawTermChange: tx.RawTermChange(),
		}
		m.messageSender.BroadcastMessage(message.MessageTypeTermChange, &msg)

	case *tx_types.Archive:
		msg := p2p_message.MessageNewArchive{
			Archive: tx,
		}
		m.messageSender.BroadcastMessage(message.MessageTypeArchive, &msg)
	case *tx_types.ActionTx:
		msg := p2p_message.MessageNewActionTx{
			ActionTx: tx,
		}
		m.messageSender.BroadcastMessage(message.MessageTypeActionTX, &msg)

	default:
		log.Warn("never come here, unknown tx type ", tx)
	}
}
