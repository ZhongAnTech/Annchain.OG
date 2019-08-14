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
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/consensus/bft"
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
	p2pM := &OGMessage{messageType: p2p_message.MessageTypeNewTx, data: data, sourceID: "123", message: msg}
	p2pM.calculateHash()
	hub.cacheMessage(p2pM)
	ids := hub.getMsgFromCache(p2p_message.MessageTypeNewTx, *p2pM.hash)
	fmt.Println(ids)
	p2pM = nil
	ids = hub.getMsgFromCache(p2p_message.MessageTypeNewTx, tx.GetTxHash())
	fmt.Println(ids)
}

func TestP2PMessage_Unmarshal(t *testing.T) {
	var p2pMsg OGMessage
	hash := common.RandomHash()
	p2pMsg.messageType = p2p_message.MessageTypePreVote
	p2pMsg.message = &bft.MessagePreVote{
		MessageConsensus: bft.MessageConsensus{
			SourceId: 10,
			HeightRound: bft.HeightRound{
				Height: 12,
				Round:  15,
			},
		},
		Signature: common.RandomAddress().ToBytes(),
		PublicKey: common.RandomHash().ToBytes(),
		Idv:       &hash,
	}
	err := p2pMsg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	//fmt.Println(p2pMsg)
	fmt.Println(p2pMsg.message)
	p2pMsg.message = nil
	p2pMsg.marshalState = false
	//fmt.Println(p2pMsg)
	fmt.Println(p2pMsg.message)
	err = p2pMsg.Unmarshal()
	if err != nil {
		t.Fatal(err)
	}
	//fmt.Println(p2pMsg)
	fmt.Println(p2pMsg.message)
}
