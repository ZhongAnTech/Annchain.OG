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
package bft_test

import (
	"fmt"
	"github.com/annchain/OG/consensus/bft"
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
	//logrus.SetReportCaller(true)

	//filenameHook := filename.NewHook()
	//filenameHook.Field = "line"
	//logrus.AddHook(filenameHook)
}

func setupPeers(good int, bad int, bf ByzantineFeatures) []bft.BftOperator {
	pg := &dummyProposalGenerator{}
	pv := &dummyProposalValidator{}
	dm := &dummyDecisionMaker{}

	var peers []bft.BftOperator
	var peerChans []chan bft.BftMessage
	var peerInfo []bft.PeerInfo

	total := good + bad
	i := 0

	// prepare incoming channels
	for ; i < total; i++ {
		peerChans = append(peerChans, make(chan bft.BftMessage, 5))
	}

	// building communication channels
	for i = 0; i < good; i++ {
		pc := NewDummyBftPeerCommunicator(i, peerChans[i], peerChans)
		pc.Run()

		peer := bft.NewDefaultBFTPartner(total, i, BlockTime, pc, pg, pv,dm)

		peers = append(peers, peer)
		peerInfo = append(peerInfo, bft.PeerInfo{Id: i})
	}
	for ; i < total; i++ {
		peer := bft.NewDefaultBFTPartner(total, i, BlockTime)
		pc := NewDummyByzantineBftPeerCommunicator(i, peerChans[i], peerChans, bf)
		pc.Run()
		peer.PeerCommunicator = pc
		peer.ProposalGenerator = pg
		peer.ProposalValidator = pv
		peer.DecisionMaker = dm
		peers = append(peers, peer)
		peerInfo = append(peerInfo, bft.PeerInfo{Id: i})
	}
	// build known peers
	for i = 0; i < total; i++ {
		peer := peers[i]
		switch peer.(type) {
		case *bft.DefaultBftPartner:
			peer.(*bft.DefaultBftPartner).BftStatus.Peers = peerInfo
		default:
			panic("not supported")
		}

	}
	return peers
}

func start(peers []bft.BftOperator, second int) {
	logrus.Info("starting")
	for _, peer := range peers {
		go bft.WaiterLoop()
		go bft.EventLoop()

	}
	time.Sleep(time.Second * 2)
	logrus.Info("starting new era")
	for _, peer := range peers {
		go bft.StartNewEra(0, 0)
		break
	}
	time.Sleep(time.Second * time.Duration(second))

	joinAllPeers(peers)
}

func joinAllPeers(peers []bft.BftOperator) {
	for {
		time.Sleep(time.Second * 2)
		for _, peer := range peers {
			bft.Stop()
		}
		fmt.Println(runtime.NumGoroutine())
		return
	}
}

func TestAllNonByzantine(t *testing.T) {
	peers := setupPeers(4, 0, ByzantineFeatures{})
	start(peers, 30)
}

func TestByzantineButOK(t *testing.T) {
	peers := setupPeers(3, 1, ByzantineFeatures{
		SilenceProposal:  true,
		SilencePreVote:   true,
		SilencePreCommit: true,
	})
	start(peers, 60)
}

func TestByzantineNotOK(t *testing.T) {
	peers := setupPeers(2, 2, ByzantineFeatures{
		SilenceProposal:  true,
		SilencePreVote:   true,
		SilencePreCommit: true,
	})
	start(peers, 60)
}

func TestBadByzantineOK(t *testing.T) {
	peers := setupPeers(2, 2, ByzantineFeatures{
		BadPreCommit: true,
		BadPreVote:   true,
		BadProposal:  true,
	})
	start(peers, 60)
}

func TestManyBadByzantineOK(t *testing.T) {
	peers := setupPeers(15, 7, ByzantineFeatures{
		BadPreCommit: true,
		BadPreVote:   true,
		BadProposal:  true,
	})
	start(peers, 60)
}

func TestGreatManyBadByzantineOK(t *testing.T) {
	peers := setupPeers(201, 100, ByzantineFeatures{
		BadPreCommit: true,
		BadPreVote:   true,
		BadProposal:  true,
	})
	start(peers, 60)
}

func TestByzantineButOKBUG(t *testing.T) {
	peers := setupPeers(3, 3, ByzantineFeatures{
		SilenceProposal:  true,
		SilencePreVote:   true,
		SilencePreCommit: true,
	})
	start(peers, 60)
}
