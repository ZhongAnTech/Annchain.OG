package dummy

import "github.com/annchain/OG/arefactor/consensus_interface"

type DummyConsensusSigner struct {
}

func (d DummyConsensusSigner) Sign(msg []byte, account consensus_interface.ConsensusAccount) consensus_interface.Signature {
	return []byte{0x88}
}
