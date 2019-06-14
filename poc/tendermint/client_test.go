package tendermint

import (
	"github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
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

	logrus.SetFormatter(Formatter)
	logrus.SetLevel(logrus.TraceLevel)

	go func() {
		logrus.Fatal(http.ListenAndServe("localhost:6060", nil))
	}()

}

func start(peers []Partner) {
	for _, peer := range peers {
		time.Sleep(2 * time.Second)
		peer.SetPeers(peers)
		peer.StartNewEra(0, 0)
		go peer.EventLoop()
	}
	for {
		time.Sleep(time.Second * 10)
	}
}

func TestAllNonByzantine(t *testing.T) {
	total := 5
	var peers []Partner
	for i := 0; i < total; i++ {
		peers = append(peers, NewPartner(total, i, BlockTime))
	}
	start(peers)
}

func TestByzantineButOK(t *testing.T) {
	total := 4
	byzantines := 1
	var peers []Partner
	for i := 0; i < total-byzantines; i++ {
		peers = append(peers, NewPartner(total, i, BlockTime))
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
	var peers []Partner
	for i := 0; i < total-byzantines; i++ {
		peers = append(peers, NewPartner(total, i, BlockTime))
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
	var peers []Partner
	for i := 0; i < total-byzantines; i++ {
		peers = append(peers, NewPartner(total, i, BlockTime))
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
	var peers []Partner
	for i := 0; i < total-byzantines; i++ {
		peers = append(peers, NewPartner(total, i, BlockTime))
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
