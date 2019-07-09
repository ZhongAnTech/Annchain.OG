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
package announcer

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types/p2p_message"
)

type MessageSender interface {
	BroadcastMessage(messageType og.MessageType, message p2p_message.Message)
	SendToAnynomous(messageType og.MessageType, msg p2p_message.Message, anyNomousPubKey *crypto.PublicKey)
	SendToPeer(peerId string, messageType og.MessageType, msg p2p_message.Message) error
}
