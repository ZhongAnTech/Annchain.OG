package tendermint

import (
	"testing"
	"time"
)

func TestAllNonByzantine(t *testing.T) {
	total := 4
	var peers []*Partner
	for i := 0; i < total; i++ {
		peers = append(peers, NewPartner(total, i))
	}

	for _, peer := range peers {
		peer.Peers = peers
		go peer.EventLoop()
		peer.StartRound(0)

	}
	for {
		time.Sleep(time.Second * 10)
	}
}
