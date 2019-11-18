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
	"github.com/annchain/OG/og/types"
	archive2 "github.com/annchain/OG/og/types/archive"
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
	case *archive2.Tx:
		msgTx := archive2.MessageNewTx{RawTx: tx.RawTx()}
		m.messageSender.BroadcastMessageWithLink(archive2.MessageTypeNewTx, &msgTx)
	case *types.Sequencer:
		msgTx := archive2.MessageNewSequencer{RawSequencer: tx.RawSequencer()}
		m.messageSender.BroadcastMessageWithLink(archive2.MessageTypeNewSequencer, &msgTx)
	case *campaign.Campaign:
		msg := types.MessageCampaign{
			RawCampaign: tx.RawCampaign(),
		}
		m.messageSender.BroadcastMessage(types.MessageTypeCampaign, &msg)
	case *campaign.TermChange:
		msg := types.MessageTermChange{
			RawTermChange: tx.RawTermChange(),
		}
		m.messageSender.BroadcastMessage(types.MessageTypeTermChange, &msg)

	case *archive.Archive:
		msg := types.MessageNewArchive{
			Archive: tx,
		}
		m.messageSender.BroadcastMessage(archive2.MessageTypeArchive, &msg)
	case *archive2.ActionTx:
		msg := archive2.MessageNewActionTx{
			ActionTx: tx,
		}
		m.messageSender.BroadcastMessage(archive2.MessageTypeActionTX, &msg)

	default:
		log.Warn("never come here, unknown tx type ", tx)
	}
}
