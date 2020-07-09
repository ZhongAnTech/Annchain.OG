package og_interface

import (
	"github.com/annchain/OG/arefactor/consensus_interface"
)

type Genesis struct {
	//RootSequencerHash og_interface.Hash
	FirstCommittee *consensus_interface.Committee
}

type GenesisStore struct {
	//RootSequencerHash string
	FirstCommittee CommitteeStore
}

type CommitteeMemberStore struct {
	PeerIndex       int
	MemberId        string
	TransportPeerId string
	PublicKey       string
}

type CommitteeStore struct {
	Version int
	Peers   []CommitteeMemberStore
}
