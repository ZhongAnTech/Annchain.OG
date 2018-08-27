package og

import (
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/p2p"
)

// Manager manages the intern msg between each OG part
type Manager struct {
	txPool         *core.TxPool
	hub            *p2p.Hub
	acquireTxQueue chan types.Hash
}
type ManagerConfig struct {
	AcquireTxQueueSize uint // length of the channel for tx acquiring
	BatchAcquireSize   uint // length of the buffer for batch tx acquire for a single node
}

func (m *Manager) Init(config ManagerConfig) {
	m.acquireTxQueue = make(chan types.Hash, config.AcquireTxQueueSize)
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

func (m *Manager) StartSync() {

}
