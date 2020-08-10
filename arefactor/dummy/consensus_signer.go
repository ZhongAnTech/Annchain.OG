package dummy

import "github.com/annchain/OG/arefactor/consensus_interface"

type DummyConsensusSigner struct {
	Id int
}

func (d DummyConsensusSigner) Sign(msg []byte, account consensus_interface.ConsensusAccount) consensus_interface.Signature {
	return []byte{byte(d.Id*10 + 8)}
}
