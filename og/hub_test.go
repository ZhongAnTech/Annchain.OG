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
