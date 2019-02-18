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

func (m *MessageRouterOG02) RouteGetNodeDataMsg(msg *p2PMessage) {
	m.GetNodeDataMsgHandler.HandleGetNodeDataMsg(msg.sourceID)
}

func (m *MessageRouterOG02) RouteNodeDataMsg(msg *p2PMessage) {
	m.NodeDataMsgHandler.HandleNodeDataMsg(msg.sourceID)
}

func (m *MessageRouterOG02) RouteGetReceiptsMsg(msg *p2PMessage) {
	m.GetReceiptsMsgHandler.HandleGetReceiptsMsg(msg.sourceID)
}
