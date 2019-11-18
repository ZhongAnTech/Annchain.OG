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

type MessageRouterOG02 struct {
	GetNodeDataMsgHandler GetNodeDataMsgHandler
	NodeDataMsgHandler    NodeDataMsgHandler
	GetReceiptsMsgHandler GetReceiptsMsgHandler
}

type GetNodeDataMsgHandler interface {
	HandleGetNodeDataMsg(peerId string)
}

type NodeDataMsgHandler interface {
	HandleNodeDataMsg(peerId string)
}

type GetReceiptsMsgHandler interface {
	HandleGetReceiptsMsg(peerId string)
}

func (m *MessageRouterOG02) Start() {
}

func (m *MessageRouterOG02) Stop() {

}

func (m *MessageRouterOG02) Name() string {
	return "MessageRouterOG32"
}

func (m *MessageRouterOG02) RouteGetNodeDataMsg(msg *types) {
	m.GetNodeDataMsgHandler.HandleGetNodeDataMsg(msg.SourceID)
}

func (m *MessageRouterOG02) RouteNodeDataMsg(msg *types) {
	m.NodeDataMsgHandler.HandleNodeDataMsg(msg.SourceID)
}

func (m *MessageRouterOG02) RouteGetReceiptsMsg(msg *types) {
	m.GetReceiptsMsgHandler.HandleGetReceiptsMsg(msg.SourceID)
}
