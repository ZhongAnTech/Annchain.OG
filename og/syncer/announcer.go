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
	"github.com/annchain/OG/consensus/campaign"
	"github.com/annchain/OG/og/archive"
	"github.com/annchain/OG/og/protocol/ogmessage"
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
func (m *Announcer) BroadcastNewTx(txi ogmessage.Txi) {
	switch tx := txi.(type) {
	case *ogmessage.Tx:
		msgTx := ogmessage.MessageNewTx{RawTx: tx.RawTx()}
		m.messageSender.BroadcastMessageWithLink(ogmessage.MessageTypeNewTx, &msgTx)
	case *ogmessage.Sequencer:
		msgTx := ogmessage.MessageNewSequencer{RawSequencer: tx.RawSequencer()}
		m.messageSender.BroadcastMessageWithLink(ogmessage.MessageTypeNewSequencer, &msgTx)
	case *campaign.Campaign:
		msg := ogmessage.MessageCampaign{
			RawCampaign: tx.RawCampaign(),
		}
		m.messageSender.BroadcastMessage(ogmessage.MessageTypeCampaign, &msg)
	case *campaign.TermChange:
		msg := ogmessage.MessageTermChange{
			RawTermChange: tx.RawTermChange(),
		}
		m.messageSender.BroadcastMessage(ogmessage.MessageTypeTermChange, &msg)

	case *archive.Archive:
		msg := ogmessage.MessageNewArchive{
			Archive: tx,
		}
		m.messageSender.BroadcastMessage(ogmessage.MessageTypeArchive, &msg)
	case *ogmessage.ActionTx:
		msg := ogmessage.MessageNewActionTx{
			ActionTx: tx,
		}
		m.messageSender.BroadcastMessage(ogmessage.MessageTypeActionTX, &msg)

	default:
		log.Warn("never come here, unknown tx type ", tx)
	}
}
