package og

import (
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/core"
	"github.com/sirupsen/logrus"
)

// Manager manages the intern msg between each OG part
type Manager struct {
	txPool         *core.TxPool
	hub            *Hub
	acquireTxQueue chan types.Hash
}

type ManagerConfig struct {
	AcquireTxQueueSize uint // length of the channel for tx acquiring
	BatchAcquireSize   uint // length of the buffer for batch tx acquire for a single node
}

func NewManager(config ManagerConfig, hub *Hub) (*Manager) {
	m := Manager{}
	m.acquireTxQueue = make(chan types.Hash, config.AcquireTxQueueSize)
	// init hub
	m.SetupCallbacks(hub)
	return &m
}

// SetupCallbacks Regist callbacks to handle different messages
func (m *Manager) SetupCallbacks(hub *Hub) {
	hub.CallbackRegistry[MessageTypePing] = m.HandleMessageTypePing
	hub.CallbackRegistry[MessageTypePong] = m.HandleMessageTypePong
	m.hub = hub
}

// EnsurePreviousTxs checks if all ancestors of the tip is in the local tx pool
// If tx is missing, send to fetching queue
// Return true if the hash's parent is there, or false if the hash's parent is missing.
func (m *Manager) EnsurePreviousTxs(tipHash types.Hash) bool {
	tx := m.txPool.Get(tipHash)
	if tx == nil {
		// needs to fetch this hash
		m.acquireTxQueue <- tipHash
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
	m.hub.Outgoing <- &P2PMessage{MessageType: MessageTypePing, Message: []byte{}}
}

func (m *Manager) Stop(){

}

func (m *Manager) Name() string{
	return "Manager"
}

func (m *Manager) HandleMessageTypePing(*P2PMessage) {
	logrus.Info("Received your ping. Respond you a pong")
	m.hub.Outgoing <- &P2PMessage{MessageType: MessageTypePong, Message: []byte{}}
}

func (m *Manager) HandleMessageTypePong(*P2PMessage) {
	logrus.Info("Received your pong.")
}
