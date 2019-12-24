package engine

import "github.com/annchain/OG/eventbus"

type LedgerProvider interface {
}

type EnginePlugin interface {
	SupportedEventHandlers() []eventbus.EventHandlerRegisterInfo
}

//type