package hotstuff

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"strconv"
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
	logrus.WithField("msg", msg).WithField("to", id).Trace("sending")
	//defer logrus.WithField("msg", msg).WithField("to", id).Trace("sent")
	h.Channels[id] <- msg
}

func (h *Hub) SendToAllButMe(msg *Msg, myId int) {
	for id := range h.Channels {
		if id != myId {
			logrus.WithField("msg", msg).WithField("to", id).Trace("broadcasting")
			//defer logrus.WithField("msg", msg).WithField("to", id).Trace("sent")
			h.Channels[id] <- msg
		}
	}
}

type Partner struct {
	MessageHub *Hub
	NodeCache  map[string]*Node
	logger     *logrus.Logger

	LockedQC    *QC
	PrepareQC   *QC
	PreCommitQC *QC
	CommitQC    *QC

	Id          int
	N           int
	F           int
	currentView int
	LeaderFunc  func(n int, currentView int) int
	Height      int

	quit chan bool
}

func (n *Partner) InitDefault() {
	n.quit = make(chan bool)
	n.currentView = 0
	n.logger = SetupOrderedLog(n.Id)
	genesis := &Node{content: "genesis"}
	n.NodeCache["genesis"] = genesis
	n.LockedQC = &QC{
		Typev:      PREPARE,
		ViewNumber: 0,
		Node:       genesis,
		Sigs:       nil,
	}
}

func (n *Partner) Start() {
	n.StartMe(0)
	n.loop()
}
func (n *Partner) loop() {
	for {
		select {
		case <-n.quit:
			return
		default:

		}
		n.currentView += 1
		n.logger.WithField("newview", n.currentView).Info("current view advanced")
		// do this first to avoid status mess
		if n.LeaderFunc(n.N, n.currentView) == n.Id {
			// I am leader in this round
			n.doLeader()
		} else {
			n.doReplica()
		}

	}
}

func (n *Partner) Stop() {
	n.logger.Info("Stopping")
	close(n.quit)
	n.logger.Info("Stopped")
}

func (n *Partner) Name() string {
	return fmt.Sprintf("Node %d", n.Id)
}

// StartMe sends a lockedQC to the current leader so that leader may trigger start.
// myCurrentView is the latest view we locked in database.
// Call this BEFORE main loop
func (n *Partner) StartMe(myCurrentView int) {
	// send NEWVIEW to the next leader
	n.currentView = myCurrentView
	n.logger.WithField("newview", n.currentView).Info("current view set")

	leaderId := n.LeaderFunc(n.N, n.currentView+1)

	n.MessageHub.Send(&Msg{
		Typev:         NEWVIEW,
		ViewNumber:    n.currentView, // I already advanced to next view, send previous view to the current leader to let him start
		Node:          n.LockedQC.Node,
		Justify:       n.LockedQC,
		FromPartnerId: n.Id,
		Sig:           Signature{},
	}, leaderId)

}

func (n *Partner) doLeader() {
	var keepGoing bool
	keepGoing = n.doLeaderPrepare()
	if keepGoing {
		keepGoing = n.doLeaderPreCommit()
	} else {
		n.logger.Warn("bad result in leader prepare")
	}
	if keepGoing {
		keepGoing = n.doLeaderCommit()
	} else {
		n.logger.Warn("bad result in leader precommit")
	}
	if keepGoing {
		keepGoing = n.doLeaderDecide()
	} else {
		n.logger.Warn("bad result in leader commit")
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
	n.PrepareQC = GenQC(goodMessages)
	n.MessageHub.SendToAllButMe(&Msg{
		Typev:         PRECOMMIT,
		ViewNumber:    n.currentView,
		Node:          nil,
		Justify:       n.PrepareQC,
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

	n.NodeCache[n.CommitQC.Node.content] = n.CommitQC.Node
	n.reachConsensus(n.CommitQC.Node)

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
	nextView := n.currentView + 1 // TODO: if the node falls behind too much, it will take a very long time to catch up.
	n.MessageHub.Send(&Msg{
		Typev:         NEWVIEW,
		ViewNumber:    n.currentView, // send previous view number to indicate that this message is for last round
		Node:          nil,
		Justify:       n.PrepareQC,
		FromPartnerId: n.Id,
		Sig:           Signature{},
	}, n.LeaderFunc(n.N, nextView)) // next round leader
	return
}

func (n *Partner) doReplica() {
	var keepGoing bool
	keepGoing = n.doReplicaPrepare()
	if keepGoing {
		keepGoing = n.doReplicaPreCommit()
	} else {
		n.logger.Warn("bad result in replica prepare")
	}
	if keepGoing {
		keepGoing = n.doReplicaCommit()
	} else {
		n.logger.Warn("bad result in replica precommit")
	}
	if keepGoing {
		keepGoing = n.doReplicaDecide()
	} else {
		n.logger.Warn("bad result in replica commit")
	}
	// finally
	n.doFinally()
}

func (n *Partner) waitForMatchingMsg(msgType MsgType, viewNumber int, waitCount int, senderId int) (collected bool, goodMessages []*Msg) {
	n.logger.WithFields(logrus.Fields{
		"Id":          n.Id,
		"waitForType": msgType,
		"waitForVN":   viewNumber,
		"myVN":        n.currentView,
		"for":         "msg",
	}).Trace("waiting")
	myChannel := n.MessageHub.GetChannel(n.Id)
	timer := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-timer.C:
			n.logger.WithFields(logrus.Fields{
				"Id":          n.Id,
				"waitForType": msgType,
				"waitForVN":   viewNumber,
				"myVN":        n.currentView,
				"for":         "msg",
			}).Warn("timeout")
			return false, nil
		case msg := <-myChannel:
			if senderId == -1 || senderId == msg.FromPartnerId {
				if MatchingMsg(msg, msgType, viewNumber) {
					// TODO: add dedup
					goodMessages = append(goodMessages, msg)
					if len(goodMessages) >= waitCount {
						collected = true
						//n.logger.WithFields(logrus.Fields{
						//	"Id":   n.Id,
						//	"type": msgType,
						//	"VN":   viewNumber,
						//	"myVN": n.currentView,
						//	"for":  "msg",
						//}).Info("collected----------")
						return
					}
				}
			} else {
				n.logger.Warn("no match 2")
			}
		}
	}
}
func (n *Partner) waitForMatchingQC(msgType MsgType, viewNumber int, waitCount int, senderId int) (collected bool, goodMessages []*Msg) {
	n.logger.WithFields(logrus.Fields{
		"Id":          n.Id,
		"waitForType": msgType,
		"waitForVN":   viewNumber,
		"myVN":        n.currentView,
		"for":         "qc",
	}).Trace("waiting")

	myChannel := n.MessageHub.GetChannel(n.Id)
	timer := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-timer.C:
			return false, nil
		case msg := <-myChannel:
			if msg.Justify == nil {
				logrus.Warn("unexpected message")
				continue
			}
			if MatchingQC(msg.Justify, msgType, viewNumber) {
				if senderId == -1 || senderId == msg.FromPartnerId {
					// TODO: add dedup
					goodMessages = append(goodMessages, msg)
					if len(goodMessages) >= waitCount {
						collected = true
						n.logger.WithFields(logrus.Fields{
							"Id":    n.Id,
							"type":  msgType,
							"viewN": viewNumber,
							"myVN":  n.currentView,
							"for":   "qc",
						}).Trace("collected ")
						return
					}
				}
			} else {
				n.logger.WithField("msg.Justify", msg.Justify).WithField("msgType", msgType).WithField("viewNumber", viewNumber).Warn("no match 1")
			}
		}
	}
}

func (n *Partner) CreateLeaf(node *Node) (newNode Node) {
	return Node{
		Previous: node.content,
		//content:  RandString(15), // get some random string,
		content: strconv.Itoa(n.Height + 1), // get some random string,
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

	n.PrepareQC = msg.Justify
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
	n.reachConsensus(msg.Justify.Node)
	return true
}

func (n *Partner) reachConsensus(node *Node) {
	n.Height, _ = strconv.Atoi(node.content)
	n.logger.WithFields(logrus.Fields{
		"Id":    n.Id,
		"ViewN": n.currentView,
		//"content": n.CommitQC.Node.content,
		"content": node.content,
	}).Info("reached consensus")
}
