// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package bft

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"runtime"
	"testing"
	"time"
)

var BlockTime = time.Millisecond * 1

func init() {
	Formatter := new(logrus.TextFormatter)
	//Formatter.ForceColors = false
	Formatter.DisableColors = true
	Formatter.TimestampFormat = "15:04:05.000000"
	Formatter.FullTimestamp = true
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(Formatter)
	//filenameHook := filename.NewHook()
	//filenameHook.Field = "line"
	//logrus.AddHook(filenameHook)
}

func setupPeers(good int, bad int, bf ByzantineFeatures) []BftOperator {
	pg := &dummyProposalGenerator{}
	pv := &dummyProposalValidator{}
	dm := &dummyDecisionMaker{}

	var peers []BftOperator
	var peerChans []chan BftMessage

	total := good + bad
	i := 0
	for ; i < good; i++ {
		peer := NewDefaultBFTPartner(total, i, BlockTime)
		pc := NewDummyBftPeerCommunicator(total, i)
		peer.PeerCommunicator = pc
		peer.ProposalGenerator = pg
		peer.ProposalValidator = pv
		peer.DecisionMaker = dm
		peers = append(peers, peer)
		peerChans = append(peerChans, pc.Incoming)
	}
	for ; i < total; i++ {
		peer := NewByzantinePartner(total, i, BlockTime, bf)
		pc := NewDummyBftPeerCommunicator(total, i)
		peer.DefaultBftOperator.PeerCommunicator = pc
		peer.DefaultBftOperator.ProposalGenerator = pg
		peer.DefaultBftOperator.ProposalValidator = pv
		peer.DefaultBftOperator.DecisionMaker = dm
		peers = append(peers, peer)
		peerChans = append(peerChans, pc.Incoming)
	}
	// build communication channel
	for i := 0; i < total; i++ {
		pc := peers[i].GetPeerCommunicator()
		pc.(*dummyBftPeerCommunicator).Peers = peerChans

	}
	return peers
}

func start(peers []BftOperator, second int) {
	for _, peer := range peers {
		peer.StartNewEra(0, 0)
		go peer.WaiterLoop()
		go peer.EventLoop()
	}
	time.Sleep(time.Second * time.Duration(second))
	joinAllPeers(peers)
}

func joinAllPeers(peers []BftOperator) {
	for {
		time.Sleep(time.Second * 2)
		for _, peer := range peers {
			peer.Stop()
		}
		fmt.Println(runtime.NumGoroutine())
		return
	}
}

func TestAllNonByzantine(t *testing.T) {
	peers := setupPeers(4, 0, ByzantineFeatures{})
	start(peers, 10)
}

func TestByzantineButOK(t *testing.T) {
	peers := setupPeers(3, 1, ByzantineFeatures{
		SilenceProposal:  true,
		SilencePreVote:   true,
		SilencePreCommit: true,
	})
	start(peers, 10)
}

func TestByzantineNotOK(t *testing.T) {
	peers := setupPeers(2, 2, ByzantineFeatures{
		SilenceProposal:  true,
		SilencePreVote:   true,
		SilencePreCommit: true,
	})
	start(peers, 10)
}

func TestBadByzantineOK(t *testing.T) {
	peers := setupPeers(2, 2, ByzantineFeatures{
		BadPreCommit: true,
		BadPreVote:   true,
		BadProposal:  true,
	})
	start(peers, 10)
}

func TestManyBadByzantineOK(t *testing.T) {
	peers := setupPeers(15, 7, ByzantineFeatures{
		BadPreCommit: true,
		BadPreVote:   true,
		BadProposal:  true,
	})
	start(peers, 10)
}


func TestByzantineButOKBUG(t *testing.T) {
	peers := setupPeers(3, 3, ByzantineFeatures{
		SilenceProposal:  true,
		SilencePreVote:   true,
		SilencePreCommit: true,
	})
	start(peers, 10)
}
