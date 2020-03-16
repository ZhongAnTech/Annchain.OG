package hotstuff

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

/**
Implemented according to
HotStuff: BFT Consensus in the Lens of Blockchain
Maofan Yin, Dahlia Malkhi, Michael K. Reiter, Guy Golan Gueta and Ittai Abraham
*/

type Hub struct {
	Channels map[int]chan *Msg
}

func (h *Hub) GetChannel(id int) chan *Msg {
	return h.Channels[id]
}

func (h *Hub) Send(msg *Msg, id int) {
	logrus.WithField("msg", msg).WithField("to", id).Info("sending")
	defer logrus.WithField("msg", msg).WithField("to", id).Info("sent")
	h.Channels[id] <- msg
}

func (h *Hub) SendToAllButMe(msg *Msg, myId int) {
	for id := range h.Channels {
		if id != myId {
			h.Send(msg, id)
		}
	}
}

type Partner struct {
	MessageHub *Hub
	NodeCache  map[string]*Node
	logger     *logrus.Logger

	LockedQC    *QC
	PreparedQC  *QC
	PreCommitQC *QC
	CommitQC    *QC

	Id          int
	N           int
	F           int
	currentView int
	LeaderFunc  func(n int, currentView int) int

	quit chan bool
}

func (n *Partner) InitDefault() {
	n.quit = make(chan bool)
	n.currentView = 1
	n.logger = SetupOrderedLog(n.Id)
}
func (n *Partner) Start() {
	for {
		select {
		case <-n.quit:
			return
		default:

		}
		if n.LeaderFunc(n.N, n.currentView) == n.Id {
			// I am leader in this round
			n.doLeader()
		} else {
			n.doReplica()
		}

	}
}

func (n *Partner) Stop() {
	panic("implement me")
}

func (n *Partner) Name() string {
	return fmt.Sprintf("Node %d", n.Id)
}

func (n *Partner) StartMe() {
	n.NodeCache["genesis"] = &Node{content: "genesis"}
	// send NEWVIEW to leader
	leaderId := n.LeaderFunc(n.N, n.currentView)
	n.MessageHub.Send(&Msg{
		Typev:      NEWVIEW,
		ViewNumber: 0,
		Node:       nil,
		Justify: &QC{
			Typev:      PREPARE,
			ViewNumber: 0,
			Node:       n.NodeCache["genesis"],
			Sigs:       nil,
		},
	}, leaderId)
}

func (n *Partner) doLeader() {
	var keepGoing bool
	keepGoing = n.doLeaderPrepare()
	if keepGoing {
		keepGoing = n.doLeaderPreCommit()
	}
	if keepGoing {
		keepGoing = n.doLeaderCommit()
	}
	if keepGoing {
		keepGoing = n.doLeaderDecide()
	}
	// finally
	n.doFinally()
}

func (n *Partner) doLeaderPrepare() bool {
	// wait for newview message
	collected, goodMessages := n.waitForMatchingMsg(NEWVIEW, n.currentView-1, n.N-n.F, -1)
	if !collected {
		return false
	}

	// there must be at least one
	var highQC *QC = goodMessages[0].Justify
	maxViewNumber := goodMessages[0].Justify.ViewNumber

	for _, msg := range goodMessages {
		if msg.Justify.ViewNumber > maxViewNumber {
			highQC = msg.Justify
			maxViewNumber = msg.Justify.ViewNumber
		}
	}
	curProposal := n.CreateLeaf(highQC.Node)
	n.MessageHub.SendToAllButMe(&Msg{
		Typev:         PREPARE,
		ViewNumber:    n.currentView,
		Node:          &curProposal,
		Justify:       highQC,
		FromPartnerId: n.Id,
	}, n.Id)
	return true
}

func (n *Partner) doLeaderPreCommit() bool {
	// wait for newview message
	collected, goodMessages := n.waitForMatchingMsg(PREPARE, n.currentView, n.N-n.F, -1)
	if !collected {
		return false
	}
	n.PreparedQC = GenQC(goodMessages)
	n.MessageHub.SendToAllButMe(&Msg{
		Typev:         PRECOMMIT,
		ViewNumber:    n.currentView,
		Node:          nil,
		Justify:       n.PreparedQC,
		FromPartnerId: n.Id,
	}, n.Id)
	return true
}

func (n *Partner) doLeaderCommit() bool {
	collected, goodMessages := n.waitForMatchingMsg(PRECOMMIT, n.currentView, n.N-n.F, -1)
	if !collected {
		return false
	}
	n.PreCommitQC = GenQC(goodMessages)
	n.MessageHub.SendToAllButMe(&Msg{
		Typev:         COMMIT,
		ViewNumber:    n.currentView,
		Node:          nil,
		Justify:       n.PreCommitQC,
		FromPartnerId: n.Id,
	}, n.Id)
	return true
}

func (n *Partner) doLeaderDecide() bool {
	collected, goodMessages := n.waitForMatchingMsg(COMMIT, n.currentView, n.N-n.F, -1)
	if !collected {
		return false
	}
	n.CommitQC = GenQC(goodMessages)
	n.MessageHub.SendToAllButMe(&Msg{
		Typev:         DECIDE,
		ViewNumber:    n.currentView,
		Node:          nil,
		Justify:       n.CommitQC,
		FromPartnerId: n.Id,
	}, n.Id)
	return true
}

func (n *Partner) doFinally() {
	// do this first to avoid status mess
	n.currentView += 1
	n.MessageHub.Send(&Msg{
		Typev:      NEWVIEW,
		ViewNumber: n.currentView - 1, // send previous view number to indicate that this message is for last round
		Node:       nil,
		Justify:    n.PreparedQC,
	}, n.LeaderFunc(n.N, n.currentView)) // next round leader
	return
}

func (n *Partner) doReplica() {
	var keepGoing bool
	keepGoing = n.doReplicaPrepare()
	if keepGoing {
		keepGoing = n.doReplicaPreCommit()
	}
	if keepGoing {
		keepGoing = n.doReplicaCommit()
	}
	if keepGoing {
		keepGoing = n.doReplicaDecide()
	}
	// finally
	n.doFinally()
}

func (n *Partner) waitForMatchingMsg(msgType MsgType, viewNumber int, waitCount int, senderId int) (collected bool, goodMessages []*Msg) {
	n.logger.WithFields(logrus.Fields{
		"Id":    n.Id,
		"type":  msgType,
		"viewN": viewNumber,
		"for":   "msg",
	}).Info("waiting")
	defer n.logger.WithFields(logrus.Fields{
		"Id":    n.Id,
		"type":  msgType,
		"viewN": viewNumber,
		"for":   "msg",
	}).Info("collected ")
	myChannel := n.MessageHub.GetChannel(n.Id)
	timer := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-timer.C:
			return false, nil
		case msg := <-myChannel:
			if senderId == -1 || senderId == msg.FromPartnerId {
				if MatchingMsg(msg, msgType, viewNumber) {
					// TODO: add dedup
					goodMessages = append(goodMessages, msg)
					if len(goodMessages) >= waitCount {
						collected = true
						return
					}
				}
			} else {
				n.logger.Info("no match 2")
			}
		}
	}
}
func (n *Partner) waitForMatchingQC(msgType MsgType, viewNumber int, waitCount int, senderId int) (collected bool, goodMessages []*Msg) {
	n.logger.WithFields(logrus.Fields{
		"Id":    n.Id,
		"type":  msgType,
		"viewN": viewNumber,
		"for":   "qc",
	}).Info("waiting")
	defer n.logger.WithFields(logrus.Fields{
		"Id":    n.Id,
		"type":  msgType,
		"viewN": viewNumber,
		"for":   "qc",
	}).Info("collected ")
	myChannel := n.MessageHub.GetChannel(n.Id)
	timer := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-timer.C:
			return false, nil
		case msg := <-myChannel:
			if MatchingQC(msg.Justify, msgType, viewNumber) {
				if senderId == -1 || senderId == msg.FromPartnerId {
					// TODO: add dedup
					goodMessages = append(goodMessages, msg)
					if len(goodMessages) >= waitCount {
						collected = true
						return
					}
				}
			} else {
				n.logger.Info("no match 1")
			}
		}
	}
}

func (n *Partner) CreateLeaf(node *Node) (newNode Node) {
	return Node{
		Previous: node.content,
		content:  RandString(15), // get some random string,
	}
}

func (n *Partner) doReplicaPrepare() bool {
	collected, msgs := n.waitForMatchingMsg(PREPARE, n.currentView, 1, n.LeaderFunc(n.N, n.currentView))
	if !collected {
		return false
	}
	msg := msgs[0]
	if IsExtends(msg.Node, msg.Justify, n.NodeCache) &&
		SafeNode(msg.Node, msg.Justify, n) {
		n.MessageHub.Send(&Msg{
			Typev:         PREPARE,
			ViewNumber:    n.currentView,
			Node:          msg.Node,
			Justify:       nil,
			FromPartnerId: n.Id,
			Sig: Signature{
				PartnerId: n.Id,
				Vote:      true,
			},
		}, n.LeaderFunc(n.N, n.currentView))
		return true
	}
	// TODO: here instantly break if I receive a wrong message from leader.
	return false

}

func (n *Partner) doReplicaPreCommit() bool {
	collected, msgs := n.waitForMatchingQC(PREPARE, n.currentView, 1, n.LeaderFunc(n.N, n.currentView))
	if !collected {
		return false
	}
	msg := msgs[0]

	n.PreparedQC = msg.Justify
	n.MessageHub.Send(&Msg{
		Typev:         PRECOMMIT,
		ViewNumber:    n.currentView,
		Node:          msg.Justify.Node,
		Justify:       nil,
		FromPartnerId: n.Id,
		Sig: Signature{
			PartnerId: n.Id,
			Vote:      true,
		},
	}, n.LeaderFunc(n.N, n.currentView))
	return true
}

func (n *Partner) doReplicaCommit() bool {
	collected, msgs := n.waitForMatchingQC(PRECOMMIT, n.currentView, 1, n.LeaderFunc(n.N, n.currentView))
	if !collected {
		return false
	}
	msg := msgs[0]

	n.LockedQC = msg.Justify
	n.MessageHub.Send(&Msg{
		Typev:         COMMIT,
		ViewNumber:    n.currentView,
		Node:          msg.Justify.Node,
		Justify:       nil,
		FromPartnerId: n.Id,
		Sig: Signature{
			PartnerId: n.Id,
			Vote:      true,
		},
	}, n.LeaderFunc(n.N, n.currentView))
	return true
}

func (n *Partner) doReplicaDecide() bool {
	collected, msgs := n.waitForMatchingQC(COMMIT, n.currentView, 1, n.LeaderFunc(n.N, n.currentView))
	if !collected {
		return false
	}
	msg := msgs[0]
	n.NodeCache[msg.Justify.Node.content] = msg.Justify.Node
	n.logger.WithFields(logrus.Fields{
		"Id":      n.Id,
		"ViewN":   n.currentView,
		"content": msg.Justify.Node.content,
	}).Info("reached consensus")
	return true
}
