package engine

import "github.com/annchain/OG/eventbus"

type LedgerProvider interface {
}

type EnginePlugin interface {
	SupportedEvents() []eventbus.EventRegisterInfo
}

type
