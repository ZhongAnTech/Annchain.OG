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
	p2pM := &p2PMessage{messageType: MessageTypeNewTx, data: data, sourceID: "123", message: msg}
	p2pM.calculateHash()
	hub.cacheMessage(p2pM)
	ids := hub.getMsgFromCache(MessageTypeNewTx, *p2pM.hash)
	fmt.Println(ids)
	p2pM = nil
	ids = hub.getMsgFromCache(MessageTypeNewTx, tx.GetTxHash())
	fmt.Println(ids)
}
