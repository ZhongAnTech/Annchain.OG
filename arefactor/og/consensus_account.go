package og

import "github.com/annchain/OG/arefactor/consensus_interface"

type BlsConsensusAccountProvider struct {
	BackFilePath string
	account      *consensus_interface.ConsensusAccount
}

func (b BlsConsensusAccountProvider) ProvideAccount() (*consensus_interface.ConsensusAccount, error) {
	panic("implement me")
}

func (b BlsConsensusAccountProvider) Generate() (account *consensus_interface.ConsensusAccount, err error) {
	panic("implement me")
}

func (b BlsConsensusAccountProvider) Load() (err error) {
	panic("implement me")
}

func (b BlsConsensusAccountProvider) Save() (err error) {
	panic("implement me")
}
