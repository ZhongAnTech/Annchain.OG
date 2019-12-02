package engine

import "github.com/annchain/OG/og/communication"

type OgMessageEventHandler interface {
	Handle(msgEvent *communication.OgMessageEvent)
}
