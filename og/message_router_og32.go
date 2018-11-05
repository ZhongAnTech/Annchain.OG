package og

type MessageRouterOG32 struct {
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

func (m *MessageRouterOG32) Start() {
}

func (m *MessageRouterOG32) Stop() {

}

func (m *MessageRouterOG32) Name() string {
	return "MessageRouterOG32"
}

func (m *MessageRouterOG32) RouteGetNodeDataMsg(msg *P2PMessage) {
	m.GetNodeDataMsgHandler.HandleGetNodeDataMsg(msg.SourceID)
}

func (m *MessageRouterOG32) RouteNodeDataMsg(msg *P2PMessage) {
	m.NodeDataMsgHandler.HandleNodeDataMsg(msg.SourceID)
}

func (m *MessageRouterOG32) RouteGetReceiptsMsg(msg *P2PMessage) {
	m.GetReceiptsMsgHandler.HandleGetReceiptsMsg(msg.SourceID)
}
