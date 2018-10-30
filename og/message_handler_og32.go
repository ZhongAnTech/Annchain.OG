package og


// IncomingMessageHandler is the default handler of all incoming messages for OG
type IncomingMessageHandlerOG32 struct {
	Og  *Og
	Hub *Hub
}

func (h *IncomingMessageHandlerOG32) HandleGetNodeDataMsg(peerId string) {
	msgLog.Warn("got GetNodeDataMsg ")
	//todo
	//p.SendNodeData(nil)
	msgLog.Debug("need send node data")
}

func (h *IncomingMessageHandlerOG32) HandleNodeDataMsg(peerId string) {
	// Deliver all to the downloader
	if err := h.Hub.Downloader.DeliverNodeData(peerId, nil); err != nil {
		msgLog.Debug("Failed to deliver node state data", "err", err)
	}
}
func (h *IncomingMessageHandlerOG32) HandleGetReceiptsMsg(peerId string) {

}
