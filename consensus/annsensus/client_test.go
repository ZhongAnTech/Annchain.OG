package annsensus

import (
	"github.com/annchain/OG/common/filename"
	"github.com/sirupsen/logrus"
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
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(Formatter)
	filenameHook := filename.NewHook()
	filenameHook.Field = "line"
	logrus.AddHook(filenameHook)
}

func start(peers []BFTPartner) {
	for _, peer := range peers {
		peer.SetPeers(peers)
		peer.StartNewEra(0, 0)
		go peer.WaiterLoop()
		go peer.EventLoop()
	}
	for {
		time.Sleep(time.Second * 5)
		return
	}
}

func TestAllNonByzantine(t *testing.T) {
	total := 22
	var peers []BFTPartner
	for i := 0; i < total; i++ {
		peers = append(peers, NewBFTPartner(total, i, BlockTime))
	}
	start(peers)
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
	start(peers)
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
	start(peers)
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
	start(peers)
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
	start(peers)
}
