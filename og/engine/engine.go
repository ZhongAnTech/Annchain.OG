package engine

import (
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/og/router"
)

type OgEngine struct {
	messageRouter router.MessageRouter
}

func NewOgEngine() {
	messageRouter := &router.NewMessageRouter()

	annsensusPlugin := annsensus.NewAnnsensusPlugin
	messageRouter.Register()
}
