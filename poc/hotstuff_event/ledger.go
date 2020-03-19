package hotstuff_event

import (
	"github.com/sirupsen/logrus"
	"strconv"
)

// Node is a block
type Node struct {
	Previous string
	content  string
}

// Stores state merkle tree.
type Ledger struct {
	Logger *logrus.Logger
	cache  map[string]Node
}

func (l *Ledger) InitDefault() {
	l.cache = make(map[string]Node)
}

// Speculate applies cmds speculatively
func (l *Ledger) Speculate(prevBlockId string, blockId string, cmds string) (executeStateId string) {
	l.cache[blockId] = Node{
		Previous: prevBlockId,
		content:  cmds,
	}
	return Hash("len" + strconv.Itoa(len(l.cache)))
}

// GetState finds the pending state for the given blockId or nil if not present
func (l *Ledger) GetState(id string) (stateId string) {
	return "1"
}

// Commit commits the pending prefix of the given blockId and prune other branches
func (l *Ledger) Commit(stateId string) {
	l.Logger.WithField("stateId", stateId).Info("ledger commit")
}
