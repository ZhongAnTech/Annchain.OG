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
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types/p2p_message"
	"testing"
)


func TestP2PMessage_Unmarshal(t *testing.T) {
	var p2pMsg og.OGMessage
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
