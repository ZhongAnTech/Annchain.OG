package og

import (
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

// Manager manages the intern msg between each OG part
type Manager struct {
	TxPool   *core.TxPool
	Hub      *Hub
	Syncer   *Syncer
	Verifier *Verifier
	Config   *ManagerConfig
	TxBuffer *TxBuffer
	Dag      *core.Dag
}

type ManagerConfig struct {
	AcquireTxQueueSize uint // length of the channel for tx acquiring
	BatchAcquireSize   uint // length of the buffer for batch tx acquire for a single node
}

// EnsurePreviousTxs checks if all ancestors of the tip is in the local tx pool
// If tx is missing, send to fetching queue
// Return true if the hash's parent is there, or false if the hash's parent is missing.
func (m *Manager) EnsurePreviousTxs(tipHash types.Hash) bool {
	tx := m.TxPool.Get(tipHash)
	if tx == nil {
		// need to fetch this hash
		m.Syncer.Enqueue(tipHash)
		return false
	}
	// no need to further fetch the ancestors. They will be checked sooner or later
	return true
}

// FinalizePrevious
func (m *Manager) FinalizePrevious(tips []types.Hash) {
	//
}

func (m *Manager) Start() {
	m.Hub.SendMessage(MessageTypePing, []byte{})
}

func (m *Manager) Stop() {

}

func (m *Manager) Name() string {
	return "Manager"
}

func (m *Manager) HandlePing(*P2PMessage) {
	logrus.Debug("received your ping. Respond you a pong")
	m.Hub.SendMessage(MessageTypePong, []byte{1})
}

func (m *Manager) HandlePong(*P2PMessage) {
	logrus.Debug("received your pong.")
}

func (m *Manager) HandleFetchByHash(msg *P2PMessage) {
	logrus.Debug("received MessageSyncRequest")
	syncRequest := types.MessageSyncRequest{}
	_, err := syncRequest.UnmarshalMsg(msg.Message)
	if err != nil {
		logrus.Debug("invalid MessageSyncRequest format")
		return
	}
	if len(syncRequest.Hashes) == 0 {
		logrus.Debug("empty MessageSyncRequest")
		return
	}

	var txs []*types.Tx
	var seqs []*types.Sequencer

	for _, hash := range syncRequest.Hashes {
		txi := m.TxPool.Get(hash)
		if txi == nil {
			txi = m.Dag.GetTx(hash)
		}
		switch tx := txi.(type) {
		case *types.Sequencer:
			seqs = append(seqs, tx)
		case *types.Tx:
			txs = append(txs, tx)
		}

	}
	syncResponse := types.MessageSyncResponse{
		Txs:        txs,
		Sequencers: seqs,
	}
	data, err := syncResponse.MarshalMsg(nil)
	if err != nil {
		logrus.Warn("failed to marshall MessageSyncResponse message")
		return
	}

	m.Hub.SendMessage(MessageTypeFetchByHashResponse, data)
}

func (m *Manager) HandleFetchByHashResponse(msg *P2PMessage) {
	logrus.Debug("received MessageSyncResponse")
	syncResponse := types.MessageSyncResponse{}
	//bytebufferd := bytes.NewBuffer(nil)
	//bytebuffers := bytes.NewBuffer(msg.Message)
	//msgp.CopyToJSON(bytebufferd, bytebuffers)
	//fmt.Println(bytebufferd.String())

	_, err := syncResponse.UnmarshalMsg(msg.Message)
	if err != nil {
		logrus.Debug("invalid MessageSyncResponse format")
		return
	}
	if (syncResponse.Txs == nil || len(syncResponse.Txs) == 0) &&
		(syncResponse.Sequencers == nil || len(syncResponse.Sequencers) == 0) {
		logrus.Debug("empty MessageSyncResponse")
		return
	}
	for _, v := range syncResponse.Txs {
		logrus.WithField("tx", v).WithField("peer", msg.SourceID).Infof("received sync response Tx")
		m.TxBuffer.AddTx(v)
	}
	for _, v := range syncResponse.Sequencers {
		logrus.WithField("seq", v).WithField("peer", msg.SourceID).Infof("received sync response seq")
		m.TxBuffer.AddTx(v)
	}
}

func (m *Manager) HandleNewTx(msg *P2PMessage) {

	newTx := types.MessageNewTx{}
	_, err := newTx.UnmarshalMsg(msg.Message)
	if err != nil {
		logrus.WithError(err).Debug("invalid MessageNewTx format")
		return
	}
	if newTx.Tx == nil {
		logrus.Debug("empty MessageNewTx")
		return
	}
	logrus.WithField("tx", newTx.Tx).Debug("received incoming new tx")
	m.TxBuffer.AddTx(newTx.Tx)
}

func (m *Manager) HandleNewSequence(msg *P2PMessage) {
	logrus.Debug("received NewSequence")
	newSq := types.MessageNewSequence{}
	_, err := newSq.UnmarshalMsg(msg.Message)
	if err != nil {
		logrus.WithError(err).Debug("invalid NewSequence format")
		return
	}
	if newSq.Sequencer == nil {
		logrus.Debug("empty NewSequence")
		return
	}
	m.TxBuffer.AddTx(newSq.Sequencer)
}
