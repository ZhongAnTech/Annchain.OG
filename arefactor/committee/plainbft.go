package committee

import (
	"errors"
	"github.com/annchain/OG/arefactor/consensus_interface"
)

type PlainBftCommitteeProvider struct {
	Version     int
	myAccount   consensus_interface.ConsensusAccount // bls consensus account
	memberIds   []string
	members     []consensus_interface.CommitteeMember // other members
	myIndex     int
	memberIdMap map[string]consensus_interface.CommitteeMember
}

func (b *PlainBftCommitteeProvider) InitCommittee(version int, peers []consensus_interface.CommitteeMember,
	myAccount consensus_interface.ConsensusAccount) {

	b.Version = version
	b.myAccount = myAccount
	b.memberIds = []string{}
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
	}
}

func (b PlainBftCommitteeProvider) AmIIn() bool {
	return b.myIndex >= 0
}

func (b PlainBftCommitteeProvider) IsIn(id string) bool {
	for _, v := range b.memberIds {
		if v == id {
			return true
		}
	}
	return false
}

func (b PlainBftCommitteeProvider) GetVersion() int {
	return b.Version
}

func (b PlainBftCommitteeProvider) GetAllMemberPeedIds() []string {
	return b.memberIds
}

func (b *PlainBftCommitteeProvider) GetAllMembers() []consensus_interface.CommitteeMember {
	return b.members
}

func (b PlainBftCommitteeProvider) GetMyPeerId() string {
	return b.memberIds[b.myIndex]
}

func (b PlainBftCommitteeProvider) GetMyPeerIndex() int {
	return b.myIndex
}

func (b PlainBftCommitteeProvider) GetLeaderPeerId(round int64) string {
	return b.memberIds[round%int64(len(b.memberIds))]
}

func (b PlainBftCommitteeProvider) GetPeerIndex(id string) (index int, err error) {
	if v, ok := b.memberIdMap[id]; ok {
		index = v.PeerIndex
	}
	err = errors.New("peer not found in committee")
	return
}

func (b PlainBftCommitteeProvider) GetThreshold() int {
	return len(b.memberIds) * 2 / 3
}

func (b PlainBftCommitteeProvider) AmILeader(round int64) bool {
	if !b.AmIIn() {
		return false
	}
	return b.GetLeaderPeerId(round) == b.memberIds[b.myIndex]
}
