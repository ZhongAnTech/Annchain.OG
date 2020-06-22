package consensus

type RoundRobinFixedCommittee struct {
	Peers   []string
	MyIndex int
}

func (r RoundRobinFixedCommittee) GetSessionId() int {
	return 0
}

func (r RoundRobinFixedCommittee) GetAllMemberPeedIds() []string {
	return r.Peers
}

func (r RoundRobinFixedCommittee) GetMyPeerId() string {
	return r.Peers[r.MyIndex]
}

func (r RoundRobinFixedCommittee) GetMyPeerIndex() int {
	return r.MyIndex
}

func (r RoundRobinFixedCommittee) GetLeaderPeerId(round int) string {
	return r.Peers[round%len(r.Peers)]
}
