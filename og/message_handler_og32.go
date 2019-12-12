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

// IncomingMessageHandler is the default handler of all incoming messages for OG
type IncomingMessageHandlerOG02 struct {
	Og  *Og
	Hub *Hub
}

func (h *IncomingMessageHandlerOG02) HandleGetNodeDataMsg(peerId string) {
	message_archive.msgLog.Warn("got GetNodeDataMsg")
	//todo
	//p.SendNodeData(nil)
	message_archive.msgLog.Debug("need send node Data")
}

func (h *IncomingMessageHandlerOG02) HandleNodeDataMsg(peerId string) {
	// Deliver all to the downloader
	if err := h.Hub.Downloader.DeliverNodeData(peerId, nil); err != nil {
		message_archive.msgLog.Debug("Failed to deliver node state Data", "err", err)
	}
}
func (h *IncomingMessageHandlerOG02) HandleGetReceiptsMsg(peerId string) {

}
