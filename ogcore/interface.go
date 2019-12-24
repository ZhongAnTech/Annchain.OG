package ogcore

import (
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/ogcore/model"
)

type EventBus interface {
	Route(eventbus.Event)
}

type OgStatusProvider interface {
	GetCurrentOgStatus() model.OgStatusData
}
