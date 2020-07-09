package committee

import (
	"errors"
	"github.com/annchain/OG/arefactor/consensus_interface"
)

// BlsCommitteeProvider holds a single round of committee.
type BlsCommitteeProvider struct {
	Version            int
	myAccount          consensus_interface.ConsensusAccount // bls consensus account
	memberIds          []string
	memberTransportIds []string
	members            []consensus_interface.CommitteeMember // other members
	myIndex            int
	memberIdMap        map[string]consensus_interface.CommitteeMember
}

func (b *BlsCommitteeProvider) InitCommittee(version int, peers []consensus_interface.CommitteeMember,
	myAccount consensus_interface.ConsensusAccount) {

	b.Version = version
	b.myAccount = myAccount
	b.memberIds = []string{}
	b.memberTransportIds = []string{}
	b.members = []consensus_interface.CommitteeMember{}
	b.memberIdMap = make(map[string]consensus_interface.CommitteeMember)
	b.myIndex = 0
	for i, peer := range peers {
		if b.myIndex == 0 && peer.MemberId == myAccount.Id() {
			b.myIndex = i
		}
		b.memberIdMap[peer.MemberId] = peer
		b.members = append(b.members, peer)
		b.memberIds = append(b.memberIds, peer.MemberId)
		b.memberTransportIds = append(b.memberTransportIds, peer.TransportPeerId)
	}
}

func (b BlsCommitteeProvider) AmIIn() bool {
	return b.myIndex >= 0
}

func (b BlsCommitteeProvider) IsIn(id string) bool {
	for _, v := range b.memberIds {
		if v == id {
			return true
		}
	}
	return false
}

func (b BlsCommitteeProvider) GetVersion() int {
	return b.Version
}

func (b BlsCommitteeProvider) GetAllMemberPeedIds() []string {
	return b.memberIds
}

func (b BlsCommitteeProvider) GetAllMemberTransportIds() []string {
	return b.memberTransportIds
}

func (b *BlsCommitteeProvider) GetAllMembers() []consensus_interface.CommitteeMember {
	return b.members
}

func (b BlsCommitteeProvider) GetMyPeerId() string {
	return b.memberIds[b.myIndex]
}

func (b BlsCommitteeProvider) GetMyPeerIndex() int {
	return b.myIndex
}

func (b BlsCommitteeProvider) GetLeader(round int64) consensus_interface.CommitteeMember {
	return b.members[round%int64(len(b.memberIds))]
}

func (b BlsCommitteeProvider) GetPeerIndex(id string) (index int, err error) {
	if v, ok := b.memberIdMap[id]; ok {
		index = v.PeerIndex
	}
	err = errors.New("peer not found in committee")
	return
}

func (b BlsCommitteeProvider) GetThreshold() int {
	return len(b.memberIds) * 2 / 3
}

func (b BlsCommitteeProvider) AmILeader(round int64) bool {
	if !b.AmIIn() {
		return false
	}
	return b.GetLeader(round).MemberId == b.memberIds[b.myIndex]
}
