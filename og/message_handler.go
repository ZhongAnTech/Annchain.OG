package og

import (
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

// IncomingMessageHandler is the default handler of all incoming messages for OG
type IncomingMessageHandler struct {
	TxBuffer *TxBuffer
	Hub      *Hub
}

func (m *IncomingMessageHandler) HandlePing() {
	logrus.Debug("received your ping. Respond you a pong")
	m.Hub.BroadcastMessage(MessageTypePong, []byte{1})
}

func (m *IncomingMessageHandler) HandlePong() {
	logrus.Debug("received your pong.")
}

func (m *IncomingMessageHandler) HandleFetchByHashResponse(syncResponse types.MessageSyncResponse, sourceID string) {
	logrus.WithField("q", syncResponse.String()).Debug("received MessageSyncResponse")

	for _, v := range syncResponse.Txs {
		logrus.WithField("tx", v).WithField("peer", sourceID).Debugf("received sync response Tx")
		m.TxBuffer.AddTx(v)
	}
	for _, v := range syncResponse.Sequencers {
		logrus.WithField("seq", v).WithField("peer", sourceID).Debugf("received sync response seq")
		m.TxBuffer.AddTx(v)
	}
}

func (m *IncomingMessageHandler) HandleNewTx(newTx types.MessageNewTx) {
	logrus.WithField("q", newTx).Debug("received MessageNewTx")
	m.TxBuffer.AddTx(newTx.Tx)
}

func (m *IncomingMessageHandler) HandleNewTxs(newTxs types.MessageNewTxs) {
	logrus.WithField("q", newTxs).Debug("received MessageNewTxs")

	for _, tx := range newTxs.Txs {
		m.TxBuffer.AddTx(tx)
	}
}

func (m *IncomingMessageHandler) HandleNewSequencer(newSeq types.MessageNewSequencer) {
	logrus.WithField("q", newSeq).Debug("received NewSequence")

	m.TxBuffer.AddTx(newSeq.Sequencer)
}
