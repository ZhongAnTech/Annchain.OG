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
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
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
	txType := txi.GetType()
	if txType == types.TxBaseTypeNormal {
		tx := txi.(*types.Tx)
		msgTx := types.MessageNewTx{RawTx: tx.RawTx()}
		m.messageSender.BroadcastMessageWithLink(og.MessageTypeNewTx, &msgTx)
	} else if txType == types.TxBaseTypeSequencer {
		seq := txi.(*types.Sequencer)
		msgTx := types.MessageNewSequencer{RawSequencer: seq.RawSequencer()}
		m.messageSender.BroadcastMessageWithLink(og.MessageTypeNewSequencer, &msgTx)
	} else {
		log.Warn("never come here, unknown tx type", txType)
	}
}
