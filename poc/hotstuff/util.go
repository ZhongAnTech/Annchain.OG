package hotstuff

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

type MsgType int

func (m MsgType) String() string {
	switch m {
	case NEWVIEW:
		return "NEWVIEW"
	case PREPARE:
		return "PREPARE"
	case PRECOMMIT:
		return "PRECOMMIT"
	case COMMIT:
		return "COMMIT"
	case DECIDE:
		return "DECIDE"
	default:
		return "NA"
	}
}

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

func (n Node) String() string {
	return "c:" + n.content
}

type Msg struct {
	Typev         MsgType
	ViewNumber    int
	Node          *Node
	Justify       *QC
	FromPartnerId int
	Sig           Signature
}

func (m Msg) String() string {
	return fmt.Sprintf("[type:%s VN:%d Node:%s JT:%s From:%d]", m.Typev, m.ViewNumber, m.Node, m.Justify, m.FromPartnerId)
}

type QC struct {
	Typev      MsgType
	ViewNumber int
	Node       *Node
	Sigs       []Signature // simulate sig by give an id to the vote
}

func (m QC) String() string {
	return fmt.Sprintf("[type:%s VN:%d Node:%s Sigs:%+v]", m.Typev, m.ViewNumber, m.Node.content, m.Sigs)

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

func SafeNode(node *Node, qc *QC, partner *Partner) (isSafe bool) {
	isExtends := IsExtends(node, partner.LockedQC, partner.NodeCache)                           // safety rule
	liveness := qc.ViewNumber > partner.LockedQC.ViewNumber || partner.LockedQC.ViewNumber == 0 // liveness rule

	isSafe = isExtends || liveness
	if !isSafe {
		if !isExtends {
			logrus.Warn("isExtends warn")
		}
		if !liveness {
			logrus.WithField("qc", qc).WithField("lockedqc", partner.LockedQC).Warn("liveness warn")
		}
	}
	if !isExtends {
		if liveness {
			logrus.Warn("liveness jump")
		}
	}
	return
}

func IsExtends(node *Node, qc *QC, cache map[string]*Node) bool {
	current := node
	for current != nil {
		if current.content == qc.Node.content {
			return true
		}
		current = cache[current.Previous]
	}
	logrus.Warn("not isextends")
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
