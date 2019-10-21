package protocol

import "github.com/annchain/OG/og/protocol/ogmessage"

// Verifier defines interface to validate a tx.
// There may be lots of rules for a tx to be valid across the project
// such as Consensus(Annsensus), Graph structure(DAG), Tx format(mining), etc
type Verifier interface {
	Verify(t ogmessage.Txi) bool
	Name() string
	String() string
	Independent() bool
}
