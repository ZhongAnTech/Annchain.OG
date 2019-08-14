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
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/annchain/OG/types/tx_types"
	"github.com/deckarep/golang-set"
	"testing"
)

func TestPeerSet_GetRandomPeers(t *testing.T) {

	set := newPeerSet()
	for i := 0; i < 10; i++ {
		rawPeer := p2p.Peer{}
		p := &peer{
			Peer:      &rawPeer,
			version:   1,
			id:        fmt.Sprintf("%d", i),
			knownMsg:  mapset.NewSet(),
			queuedMsg: make(chan []*OGMessage, maxqueuedMsg),
			term:      make(chan struct{}),
		}
		set.Register(p)
	}
	fmt.Println("len peers", set.Len())
	peers := set.GetRandomPeers(3)
	if len(peers) != 3 {
		t.Fatalf("peers size mismatch, wanted 3 ,got %d ,peers %v ", len(peers), peers)
	}
	peers = set.GetPeers(nil, 3)
	if len(peers) != 0 {
		t.Fatalf("peers size mismatch, wanted 3 ,got %d ,peers %v", len(peers), peers)
	}

}

func TestPeer_MarkMessage(t *testing.T) {
	rawPeer := p2p.Peer{}
	p := &peer{
		Peer:      &rawPeer,
		version:   1,
		id:        "111",
		knownMsg:  mapset.NewSet(),
		queuedMsg: make(chan []*OGMessage, maxqueuedMsg),
		term:      make(chan struct{}),
	}
	var msgs []OGMessage
	for i := 0; i < 100; i++ {
		msg := OGMessage{message: &p2p_message.MessageNewTx{RawTx: tx_types.RandomTx().RawTx()}, messageType: p2p_message.MessageTypeNewTx}
		msg.Marshal()
		msg.calculateHash()
		msgs = append(msgs, msg)
		p.MarkMessage(msg.messageType, *msg.hash)
	}

	for i, val := range msgs {
		key := val.msgKey()
		if !p.knownMsg.Contains(key) {
			t.Fatal(i, val, key)
		}
	}

}
