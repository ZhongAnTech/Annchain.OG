package og

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
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
		{0, downloader.FullSync, true}, {1, downloader.FullSync, true}, {2, downloader.FullSync, true},
		{0, downloader.FastSync, false}, {1, downloader.FastSync, false}, {2, downloader.FastSync, true},
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

func TestP2PMessage_Encrypt(t *testing.T) {
	for i:=0;i<2;i++ {
		logrus.SetLevel(logrus.TraceLevel)
		msg := types.MessageConsensusDkgDeal{
			Data: []byte("this is a test of og message"),
			Id:   12,
		}
		m := p2PMessage{message: &msg, messageType: MessageTypeConsensusDkgDeal}
		s := crypto.NewSigner(crypto.CryptoType(i))
		fmt.Println(s.GetCryptoType())
		pk, sk, _ := s.RandomKeyPair()
		m.Marshal()
		logrus.Debug(len(m.data))
		err := m.Encrypt(&pk)
		if err != nil {
			t.Fatal(err)
		}
		logrus.Debug(len(m.data))
		mm := p2PMessage{data: m.data, messageType: MessageTypeSecret}
		ok := mm.checkRequiredSize()
		logrus.Debug(ok)
		ok = mm.maybeIsforMe(&pk)
		if !ok {
			t.Fatal(ok)
		}
		err = mm.Decrypt(&sk)
		if err != nil {
			t.Fatal(err)
		}
		logrus.Debug(len(mm.data),mm.messageType)
		err = mm.Unmarshal()
		if err != nil {
			t.Fatal(err)
		}
		logrus.Debug(len(mm.data))
		dkgMsg:= mm.message.(*types.MessageConsensusDkgDeal)
		logrus.Debug(dkgMsg.Id," " ,string(dkgMsg.Data))
		logrus.Debug(mm.message)
	}
}
