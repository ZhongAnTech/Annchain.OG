package plugin_test

import (
	"github.com/annchain/OG/message"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/plugin/og"
	"testing"
	"time"
)

func TestOgPartner(t *testing.T) {
	len := 2
	plugins := make([]*og.OgPlugin, len)
	chans := make([]chan *message.GeneralMessageEvent, len)
	communicators := make([]*LocalGeneralPeerCommunicator, len)

	for i := 0; i < len; i++ {
		chans[i] = make(chan *message.GeneralMessageEvent)
	}

	for i := 0; i < len; i++ {
		communicators[i] = NewLocalGeneralPeerCommunicator(i, chans[i], chans)
	}

	for i := 0; i < len; i++ {
		plugins[i] = og.NewOgPlugin(communicators[i])
		plugins[i].Start()
	}

	// init general processor

	plugins[0].OgPartner.SendMessagePing(communication.OgPeer{Id: 1})
	time.Sleep(time.Minute)
}
