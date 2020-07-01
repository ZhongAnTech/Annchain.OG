package committee

import "errors"

// BlsCommitteeProvider holds a single round of committee.
type BlsCommitteeProvider struct {
	SessionId       int
	MemberPeerIds   []string
	MyIndex         int
	MemberPeerIdMap map[string]int
}

func (b *BlsCommitteeProvider) InitCommittee(peerIds []string, myIndex int) {
	b.MemberPeerIds = peerIds
	b.MemberPeerIdMap = make(map[string]int)
	for i, v := range peerIds {
		b.MemberPeerIdMap[v] = i
	}
	b.MyIndex = myIndex
}

func (b BlsCommitteeProvider) AmIIn() bool {
	return b.MyIndex >= 0
}

func (b BlsCommitteeProvider) IsIn(id string) bool {
	for _, v := range b.MemberPeerIds {
		if v == id {
			return true
		}
	}
	return false
}

func (b BlsCommitteeProvider) GetSessionId() int {
	return b.SessionId
}

func (b BlsCommitteeProvider) GetAllMemberPeedIds() []string {
	return b.MemberPeerIds
}

func (b BlsCommitteeProvider) GetMyPeerId() string {
	return b.MemberPeerIds[b.MyIndex]
}

func (b BlsCommitteeProvider) GetMyPeerIndex() int {
	return b.MyIndex
}

func (b BlsCommitteeProvider) GetLeaderPeerId(round int) string {
	return b.MemberPeerIds[round%len(b.MemberPeerIds)]
}

func (b BlsCommitteeProvider) GetPeerIndex(id string) (index int, err error) {
	if v, ok := b.MemberPeerIdMap[id]; ok {
		index = v
	}
	err = errors.New("peer not found in committee")
	return
}

func (b BlsCommitteeProvider) GetThreshold() int {
	return len(b.MemberPeerIds) * 2 / 3
}

func (b BlsCommitteeProvider) AmILeader(round int) bool {
	if !b.AmIIn() {
		return false
	}
	return b.GetLeaderPeerId(round) == b.MemberPeerIds[b.MyIndex]
}
