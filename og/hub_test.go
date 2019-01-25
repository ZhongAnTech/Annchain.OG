package og

import (
	"fmt"
	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/types"
	"testing"
	"time"
)

// Tests that protocol versions and modes of operations are matched up properly.
func TestProtocolCompatibility(t *testing.T) {

	// Define the compatibility chart
	tests := []struct {
		version    uint
		mode       downloader.SyncMode
		compatible bool
	}{
		{30, downloader.FullSync, true}, {31, downloader.FullSync, true}, {32, downloader.FullSync, true},
		{30, downloader.FastSync, false}, {31, downloader.FastSync, false}, {32, downloader.FastSync, true},
	}
	// Make sure anything we screw up is restored
	backup := ProtocolVersions
	defer func() { ProtocolVersions = backup }()

	// Try all available compatibility configs and check for errors
	for i, tt := range tests {
		ProtocolVersions = []uint{tt.version}
		h, _, err := newTestHub(tt.mode)
		if h != nil {
			defer h.Stop()
		}
		if (err == nil && !tt.compatible) || (err != nil && tt.compatible) {
			t.Errorf("test %d: compatibility mismatch: have error %v, want compatibility %v tt %v", i, err, tt.compatible, tt)
		}
	}
}

func TestSh256(t *testing.T) {
	var msg []p2PMessage
	for i := 0; i < 10000; i++ {
		var m p2PMessage
		m.messageType = MessageTypeBodiesResponse
		h := types.RandomHash()
		m.data = append(m.data, h.Bytes[:]...)
		msg = append(msg, m)
	}
	start := time.Now()
	for _, m := range msg {
		m.calculateHash()
	}
	fmt.Println("used time ", time.Now().Sub(start))
}
