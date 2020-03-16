package hotstuff

type MsgType int

const (
	NEWVIEW MsgType = iota
	PREPARE
	PRECOMMIT
	COMMIT
	DECIDE
)

// Node is a block
type Node struct {
	Previous string
	content  string
}

type Msg struct {
	Typev         MsgType
	ViewNumber    int
	Node          *Node
	Justify       *QC
	FromPartnerId int
	Sig           Signature
}

type QC struct {
	Typev      MsgType
	ViewNumber int
	Node       *Node
	Sigs       []Signature // simulate sig by give an id to the vote
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
