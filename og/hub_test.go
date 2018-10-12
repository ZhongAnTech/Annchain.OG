package og

import (
	"github.com/annchain/OG/og/downloader"
	"testing"
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
