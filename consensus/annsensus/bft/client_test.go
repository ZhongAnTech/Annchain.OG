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
	"github.com/annchain/OG/common/filename"
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
	filenameHook := filename.NewHook()
	filenameHook.Field = "line"
	logrus.AddHook(filenameHook)
}

func start(peers []BFTPartner, second int) {
	for _, peer := range peers {
		peer.SetPeers(peers)
		peer.StartNewEra(0, 0)
		go peer.WaiterLoop()
		go peer.EventLoop()
	}
	for {
		time.Sleep(time.Second * time.Duration(second))
		for _, peer := range peers {
			peer.Stop()
		}
		fmt.Println(runtime.NumGoroutine())
		return
	}
}

func TestAllNonByzantine(t *testing.T) {
	total := 4
	var peers []BFTPartner
	for i := 0; i < total; i++ {
		peers = append(peers, NewBFTPartner(total, i, BlockTime))
	}
	start(peers, 10)
}

func TestByzantineButOK(t *testing.T) {
	total := 4
	byzantines := 1
	var peers []BFTPartner
	for i := 0; i < total-byzantines; i++ {
		peers = append(peers, NewBFTPartner(total, i, BlockTime))
	}
	for i := total - byzantines; i < total; i++ {
		peers = append(peers, NewByzantinePartner(total, i, BlockTime,
			ByzantineFeatures{
				SilenceProposal:  true,
				SilencePreVote:   true,
				SilencePreCommit: true,
			}))
	}
	start(peers, 10)
}

func TestByzantineNotOK(t *testing.T) {
	total := 4
	byzantines := 2
	var peers []BFTPartner
	for i := 0; i < total-byzantines; i++ {
		peers = append(peers, NewBFTPartner(total, i, BlockTime))
	}
	for i := total - byzantines; i < total; i++ {
		peers = append(peers, NewByzantinePartner(total, i, BlockTime,
			ByzantineFeatures{
				//SilenceProposal: true,
				SilencePreVote: true,
				//SilencePreCommit: true,
			}))
	}
	start(peers, 60)
}

func TestBadByzantineOK(t *testing.T) {
	total := 4
	byzantines := 1
	var peers []BFTPartner
	for i := 0; i < total-byzantines; i++ {
		peers = append(peers, NewBFTPartner(total, i, BlockTime))
	}
	for i := total - byzantines; i < total; i++ {
		peers = append(peers, NewByzantinePartner(total, i, BlockTime,
			ByzantineFeatures{
				BadPreCommit: true,
				BadPreVote:   true,
				BadProposal:  true,
			}))
	}
	start(peers, 10)
}

func TestManyBadByzantineOK(t *testing.T) {
	total := 22
	byzantines := 7
	var peers []BFTPartner
	for i := 0; i < total-byzantines; i++ {
		peers = append(peers, NewBFTPartner(total, i, BlockTime))
	}
	for i := total - byzantines; i < total; i++ {
		peers = append(peers, NewByzantinePartner(total, i, BlockTime,
			ByzantineFeatures{
				BadPreCommit: true,
				BadPreVote:   true,
				BadProposal:  true,
			}))
	}
	start(peers, 10)
}

func TestBFT_GetInfo(t *testing.T) {
	total := 1
	var peers []BFTPartner
	for i := 0; i < total; i++ {
		peers = append(peers, NewBFTPartner(total, i, BlockTime*1000))
	}
	start(peers, 3)
}


func TestByzantineButOKBUG(t *testing.T) {
	total := 6
	byzantines := 3
	var peers []BFTPartner
	for i := 0; i < total-byzantines; i++ {
		peers = append(peers, NewBFTPartner(total, i, BlockTime))
	}
	for i := total - byzantines; i < total; i++ {
		peers = append(peers, NewByzantinePartner(total, i, BlockTime,
			ByzantineFeatures{
				SilenceProposal:  true,
				SilencePreVote:   true,
				SilencePreCommit: true,
			}))
	}
	start(peers, 10)
}
