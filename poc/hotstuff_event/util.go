package hotstuff_event

type MsgType int

const (
	PROPOSAL MsgType = iota
	VOTE
	TIMEOUT
	LOCAL_TIMEOUT
)

// Node is a block
type Node struct {
	Previous string
	content  string
}

type Msg struct {
	Typev    MsgType
	Sig      Signature
	SenderId int
	Content  Content
	//ParentQC *QC
	//Round    int
	//Id       int //

	//ViewNumber    int
	//Node          *Node
	//Justify       *QC
	//FromPartnerId int
}
type Block struct {
	Round    int
	Payload  string
	ParentQC *QC
	Id       string // hash of the previous 3 fields
}

type Content interface {
	IContent()
}

type ContentProposal struct {
	Proposal Block
}

func (c ContentProposal) IContent() {
	panic("implement me")
}

type ContentTimeout struct {
	Round int
}

func (c ContentTimeout) IContent() {
	panic("implement me")
}

type ContentVote struct {
}

func (c ContentVote) IContent() {
	panic("implement me")
}

type ContentLocalTimeout struct {
}

func (c ContentLocalTimeout) IContent() {
	panic("implement me")
}

type LedgerCommitInfo struct {
	CommitStateId *int
}

type QC struct {
	LedgerCommitInfo LedgerCommitInfo
	GrandParentId    int
	Round            int

	//Typev      MsgType
	//ViewNumber int
	//Node       *Node
	//Sigs       []Signature // simulate sig by give an id to the vote
}

type Signature struct {
	PartnerId int
	Vote      bool
}

func MatchingMsg(msg *Msg, typev MsgType, viewNumber int) bool {
	return msg.Typev == typev && msg.ViewNumber == viewNumber
}

func MatchingQC(qc *QC, typev MsgType, viewNumber int) bool {
	return qc.Typev == typev && qc.ViewNumber == viewNumber
}

func SafeNode(node *Node, qc *QC, partner *Partner) bool {
	return IsExtends(node, partner.LockedQC, partner.NodeCache) && // safety rule
		qc.ViewNumber > partner.LockedQC.ViewNumber // liveness rule
}

func IsExtends(node *Node, qc *QC, cache map[string]*Node) bool {
	current := node
	for current != nil {
		if current.content == qc.Node.content {
			return true
		}
		current = cache[current.Previous]
	}
	return false
}

func GenQC(msgs []*Msg) *QC {
	var sigs []Signature
	for _, msg := range msgs {
		sigs = append(sigs, Signature{
			PartnerId: msg.FromPartnerId,
			Vote:      true,
		})
	}
	return &QC{
		Typev:      msgs[0].Typev,
		ViewNumber: msgs[0].ViewNumber,
		Node:       msgs[0].Node,
		Sigs:       sigs,
	}
}
