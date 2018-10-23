package og

import (
	"github.com/sirupsen/logrus"
)

// IncomingMessageHandler is the default handler of all incoming messages for OG
type IncomingMessageHandlerOG32 struct {
	Og  *Og
	Hub *Hub
}

func (h *IncomingMessageHandlerOG32) HandleGetNodeDataMsg(peerId string) {
	logrus.Warn("got GetNodeDataMsg ")
	//todo
	//p.SendNodeData(nil)
	logrus.Debug("need send node data")
}

func (h *IncomingMessageHandlerOG32) HandleNodeDataMsg(peerId string) {
	// Deliver all to the downloader
	if err := h.Og.downloader.DeliverNodeData(peerId, nil); err != nil {
		logrus.Debug("Failed to deliver node state data", "err", err)
	}
}
func (h *IncomingMessageHandlerOG32) HandleGetReceiptsMsg(peerId string) {

}