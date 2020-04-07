package hotstuff_event

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
)

type DummySyncer struct {
}

// Fetch returns a block based on blockId
// it can be bruteforced. since block content is only
func (d *DummySyncer) Fetch(blockId string) {

}

// Node is a block with state
type Node struct {
	Previous string
	content  string
	total    int
	state    string
}

func (n Node) String() string {
	return fmt.Sprintf("%s %s %d", n.Previous, n.content, n.total)
}

// Stores state merkle tree.
type Ledger struct {
	Logger *logrus.Logger
	cache  map[string]Node
	mu     sync.RWMutex
}

func (l *Ledger) InitDefault() {
	l.cache = make(map[string]Node)
}

// Speculate applies cmds speculatively
func (l *Ledger) Speculate(prevBlockId string, blockId string, cmds string) (executeStateId string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, err := strconv.Atoi(cmds)
	if err != nil {
		panic(err)
	}
	if prevBlockId != "" {
		if _, ok := l.cache[prevBlockId]; !ok {
			l.Logger.WithField("missing block", prevBlockId).Warn("I'm behind. Previous block not found")
			// simulate a syncer
		}
	}

	n := Node{
		Previous: prevBlockId,
		content:  cmds,
		total:    l.cache[prevBlockId].total + v,
	}
	n.state = Hash(n.String())
	l.cache[blockId] = n

	return l.cache[blockId].state
}

// GetState finds the pending state for the given blockId or nil if not present
func (l *Ledger) GetState(id string) (stateId string) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.cache[id].state
}

// Commit commits the pending prefix of the given blockId and prune other branches
func (l *Ledger) Commit(stateId string) {
	l.Logger.WithField("stateId", stateId).Debug("ledger commit")
}
