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
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/og/message"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/annchain/OG/types/tx_types"
	"github.com/annchain/gcache"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

// Tests that protocol versions and modes of operations are matched up properly.
func TestProtocolCompatibility(t *testing.T) {

	// Define the compatibility chart
	tests := []struct {
		version    uint32
		mode       downloader.SyncMode
		compatible bool
	}{
		{0, downloader.FullSync, true}, {1, downloader.FullSync, true}, {2, downloader.FullSync, true},
		{0, downloader.FastSync, false}, {1, downloader.FastSync, false}, {2, downloader.FastSync, true},
	}
	// Make sure anything we screw up is restored
	backup := ProtocolVersions
	defer func() { ProtocolVersions = backup }()

	// Try all available compatibility configs and check for errors
	for i, tt := range tests {
		ProtocolVersions = []uint32{tt.version}
		h, _, err := newTestHub(tt.mode)
		if h != nil {
			defer h.Stop()
		}
		if (err == nil && !tt.compatible) || (err != nil && tt.compatible) {
			t.Errorf("test %d: compatibility mismatch: have error %v, want compatibility %v tt %v", i, err, tt.compatible, tt)
		}
	}
}

func TestSh256(t *testing.T) {
	var msg []OGMessage
	for i := 0; i < 10000; i++ {
		var m OGMessage
		m.messageType = message.MessageTypeBodiesResponse
		h := common.RandomHash()
		m.data = append(m.data, h.Bytes[:]...)
		msg = append(msg, m)
	}
	start := time.Now()
	for _, m := range msg {
		m.calculateHash()
	}
	fmt.Println("used time ", time.Now().Sub(start))
}

func TestP2PMessage_Encrypt(t *testing.T) {
	for i := 0; i < 2; i++ {
		logrus.SetLevel(logrus.TraceLevel)
		msg := p2p_message.MessageConsensusDkgDeal{
			Data: []byte("this is a test of og message"),
			Id:   12,
		}
		m := OGMessage{message: &msg, messageType: message.MessageTypeConsensusDkgDeal}
		s := crypto.NewSigner(crypto.CryptoType(i))
		fmt.Println(s.GetCryptoType())
		pk, sk := s.RandomKeyPair()
		m.Marshal()
		logrus.Debug(len(m.data))
		err := m.Encrypt(&pk)
		if err != nil {
			t.Fatal(err)
		}
		logrus.Debug(len(m.data))
		mm := OGMessage{data: m.data, messageType: message.MessageTypeSecret}
		ok := mm.checkRequiredSize()
		logrus.Debug(ok)
		ok = mm.maybeIsforMe(&pk)
		if !ok {
			t.Fatal(ok)
		}
		err = mm.Decrypt(&sk)
		if err != nil {
			t.Fatal(err)
		}
		logrus.Debug(len(mm.data), mm.messageType)
		err = mm.Unmarshal()
		if err != nil {
			t.Fatal(err)
		}
		logrus.Debug(len(mm.data))
		dkgMsg := mm.message.(*p2p_message.MessageConsensusDkgDeal)
		logrus.Debug(dkgMsg.Id, " ", string(dkgMsg.Data))
		logrus.Debug(mm.message)
	}
}


func TestCache(t *testing.T) {
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
	p2pM := &OGMessage{messageType: message.MessageTypeNewTx, data: data, sourceID: "123", message: msg}
	p2pM.calculateHash()
	hub.cacheMessage(p2pM)
	ids := hub.getMsgFromCache(message.MessageTypeNewTx, *p2pM.hash)
	fmt.Println(ids)
	p2pM = nil
	ids = hub.getMsgFromCache(message.MessageTypeNewTx, tx.GetTxHash())
	fmt.Println(ids)
}