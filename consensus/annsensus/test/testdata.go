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
package test

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types/p2p_message"
	"golang.org/x/crypto/sha3"
)

//go:generate msgp
//msgp:tuple TestMsg
type TestMsg struct {
	Message     p2p_message.Message
	MessageType p2p_message.MessageType
	From        common.Address
}

func (t *TestMsg) GetHash() common.Hash {
	//from := byte(t.From)
	data, err := t.Message.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	//data = append(data, from)
	h := sha3.New256()
	h.Write(data)
	b := h.Sum(nil)
	hash := common.Hash{}
	hash.MustSetBytes(b, common.PaddingNone)
	return hash
}

func (t TestMsg) String() string {
	return fmt.Sprintf("from %s, type %s, msg %s", t.From.String(), t.MessageType, t.Message)
}
