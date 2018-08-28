package og

import (
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/core"
	"github.com/sirupsen/logrus"
)

// Manager manages the intern msg between each OG part
type Manager struct {
	TxPool   *core.TxPool
	Hub      *Hub
	Syncer   *Syncer
	Verifier *Verifier
	Config   *ManagerConfig
}

type ManagerConfig struct {
	AcquireTxQueueSize uint // length of the channel for tx acquiring
	BatchAcquireSize   uint // length of the buffer for batch tx acquire for a single node
}

func NewManager(config *ManagerConfig) (*Manager) {
	m := Manager{}
	m.Config = config
	return &m
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
	m.Syncer.Enqueue(types.HexToHash("0x00"))
	m.Syncer.Enqueue(types.HexToHash("0x01"))
}

func (m *Manager) Stop() {

}

func (m *Manager) Name() string {
	return "Manager"
}

func (m *Manager) HandlePing(*P2PMessage) {
	logrus.Debug("Received your ping. Respond you a pong")
	m.Hub.outgoing <- &P2PMessage{MessageType: MessageTypePong, Message: []byte{}}
}

func (m *Manager) HandlePong(*P2PMessage) {
	logrus.Debug("Received your pong.")
}

func (m *Manager) HandleFetchByHash(msg *P2PMessage) {
	logrus.Debug("Received MessageSyncRequest")
	syncRequest := types.MessageSyncRequest{}
	_, err := syncRequest.UnmarshalMsg(msg.Message)
	if err != nil {
		logrus.Debug("Invalid MessageSyncRequest format")
		return
	}
	if len(syncRequest.Hashes) == 0 {
		logrus.Debug("Empty MessageSyncRequest")
		return
	}

	var txs []*types.Tx
	var seqs []*types.Sequencer

	for _, hash := range syncRequest.Hashes {
		// DUMMY CODE
		switch hash.Bytes[0] {
		case 0:
			tx := types.SampleSequencer()
			tx.SetHash(tx.Hash())
			seqs = append(seqs, tx)
		case 1:
			tx := types.SampleTx()
			tx.SetHash(tx.Hash())
			txs = append(txs, tx)
		}
	}
	syncResponse := types.MessageSyncResponse{
		Txs:       txs,
		Sequencer: seqs,
	}
	data, err := syncResponse.MarshalMsg(nil)
	if err != nil {
		logrus.Warn("Failed to marshall MessageSyncResponse message")
		return
	}

	m.Hub.SendMessage(MessageTypeFetchByHashResponse, data)
}

func (m *Manager) HandleFetchByHashResponse(msg *P2PMessage) {
	logrus.Debug("Received MessageSyncResponse")
	syncResponse := types.MessageSyncResponse{}
	//bytebufferd := bytes.NewBuffer(nil)
	//bytebuffers := bytes.NewBuffer(msg.Message)
	//msgp.CopyToJSON(bytebufferd, bytebuffers)
	//fmt.Println(bytebufferd.String())

	_, err := syncResponse.UnmarshalMsg(msg.Message)
	if err != nil {
		logrus.Debug("Invalid MessageSyncResponse format")
		return
	}
	if len(syncResponse.Txs) == 0 && len(syncResponse.Sequencer) == 0 {
		logrus.Debug("Empty MessageSyncResponse")
		return
	}
	for _, v := range syncResponse.Txs {
		logrus.Infof("Received Tx: %s", v.Hash().Hex())
		logrus.Infof(v.String())
	}
	for _, v := range syncResponse.Sequencer {
		logrus.Infof("Received Seq: %s", v.Hash().Hex())
		logrus.Infof(v.String())
	}
}
