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
package og

import (
	"fmt"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/annchain/OG/types/tx_types"
	"github.com/annchain/gcache"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	msgLog = logrus.StandardLogger()
	msgLog.SetLevel(logrus.TraceLevel)
	config := DefaultHubConfig()
	hub := &Hub{
		messageCache: gcache.New(config.MessageCacheMaxSize).LRU().
			Expiration(time.Second * time.Duration(config.MessageCacheExpirationSeconds)).Build(),
	}

	tx := tx_types.SampleTx()
	msg := &p2p_message.MessageNewTx{
		RawTx: tx.RawTx(),
	}
	data, _ := msg.MarshalMsg(nil)
	p2pM := &p2PMessage{messageType: MessageTypeNewTx, data: data, sourceID: "123", message: msg}
	p2pM.calculateHash()
	hub.cacheMessage(p2pM)
	ids := hub.getMsgFromCache(MessageTypeNewTx, *p2pM.hash)
	fmt.Println(ids)
	p2pM = nil
	ids = hub.getMsgFromCache(MessageTypeNewTx, tx.GetTxHash())
	fmt.Println(ids)
}
