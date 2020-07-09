package og

import "github.com/annchain/OG/arefactor/consensus_interface"

type BlsConsensusAccountProvider struct {
	BackFilePath string
	account      consensus_interface.ConsensusAccount
}

func (b *BlsConsensusAccountProvider) ProvideAccount() (consensus_interface.ConsensusAccount, error) {
	if b.account == nil {
		return b.Load()
	}
	return b.account, nil
}

func (b *BlsConsensusAccountProvider) Generate() (account consensus_interface.ConsensusAccount, err error) {
	// how to generate a bls key?
	// by discuss?
	panic("implement me")
}

func (b *BlsConsensusAccountProvider) Load() (account consensus_interface.ConsensusAccount, err error) {
	panic("implement me")
}

func (b *BlsConsensusAccountProvider) Save() (err error) {
	panic("implement me")
}
