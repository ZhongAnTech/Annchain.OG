package og

import (
	"fmt"
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	msgLog = logrus.StandardLogger()
	msgLog.SetLevel(logrus.TraceLevel)
	config := DefaultHubConfig()
	hub := &Hub{
		messageCache: gcache.New(config.MessageCacheMaxSize).LRU().
			Expiration(time.Second * time.Duration(config.MessageCacheExpirationSeconds)).Build(),
	}

	tx := types.SampleTx()
	msg := &types.MessageNewTx{
		RawTx: tx.RawTx(),
	}
	data, _ := msg.MarshalMsg(nil)
	p2pM := &P2PMessage{MessageType: MessageTypeNewTx, data: data, SourceID: "123"}
	hub.checkMsg(p2pM)
	ids := hub.getMsgFromCache(p2pM.hash)
	fmt.Println(ids)
	p2pM = nil
	ids = hub.getMsgFromCache(tx.GetTxHash())
	fmt.Println(ids)
}
