package og

import (
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/annchain/OG/arefactor/og_interface"
)

type Genesis struct {
	RootSequencerHash og_interface.Hash
	FirstCommittee    *consensus_interface.Committee
}

type CommitteeMemberStore struct {
	PeerIndex int
	MemberId  string
	PublicKey string
}

type CommitteeStore struct {
	Version int
	Peers   []CommitteeMemberStore
}

type GenesisStore struct {
	RootSequencerHash string
	FirstCommittee    CommitteeStore
}
