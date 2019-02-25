package tendermint

import (
	"testing"
	"time"
	"github.com/sirupsen/logrus"
)

func init(){
	Formatter := new(logrus.TextFormatter)
	//Formatter.ForceColors = false
	Formatter.DisableColors = true
	Formatter.TimestampFormat = "2006-01-02 15:04:05.000000"
	Formatter.FullTimestamp = true

	logrus.SetFormatter(Formatter)
}

func start(peers []Partner){
	for _, peer := range peers {
		peer.SetPeers(peers)
		go peer.EventLoop()
		peer.StartRound(0)
	}
	for {
		time.Sleep(time.Second * 10)
	}
}

func TestAllNonByzantine(t *testing.T) {
	total := 4
	var peers []Partner
	for i := 0; i < total; i++ {
		peers = append(peers, NewPartner(total, i))
	}
	start(peers)
}

func TestByzantineButOK(t *testing.T) {
	total := 4
	byzantines := 1
	var peers []Partner
	for i := 0; i < total-byzantines; i++ {
		peers = append(peers, NewPartner(total, i))
	}
	for i := total - byzantines; i < total; i++ {
		peers = append(peers, NewByzantinePartner(total, i, ByzantineFeatures{
			SilenceProposal: true,
			SilencePreVote: true,
			SilencePreCommit: true,
		}))
	}
	start(peers)
}

func TestByzantineNotOK(t *testing.T) {
	total := 4
	byzantines := 2
	var peers []Partner
	for i := 0; i < total-byzantines; i++ {
		peers = append(peers, NewPartner(total, i))
	}
	for i := total - byzantines; i < total; i++ {
		peers = append(peers, NewByzantinePartner(total, i, ByzantineFeatures{
			//SilenceProposal: true,
			SilencePreVote: true,
			//SilencePreCommit: true,
		}))
	}
	start(peers)
}