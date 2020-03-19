package hotstuff_event

// Stores state merkle tree.
type Ledger struct {
}

// Speculate applies cmds speculatively
func (l *Ledger) Speculate(prevBlockId string, blockId string, cmds string) (executeStateId string) {

}

// GetState finds the pending state for the given blockId or nil if not present
func (l *Ledger) GetState(id string) (stateId string) {

}

// Commit commits the pending prefix of the given blockId and prune other branches
func (l *Ledger) Commit(blockId string) {

}
