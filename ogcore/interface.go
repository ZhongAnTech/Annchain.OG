package ogcore

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/model"
)

type EventBus interface {
	Route(eventbus.Event)
}

type OgStatusProvider interface {
	GetCurrentOgStatus() model.OgStatusData
}

type PoolHashLocator interface {
	IsLocalHash(hash common.Hash) bool
	Get(hash common.Hash) types.Txi
}

type LedgerHashLocator interface {
	GetTx(hash common.Hash) types.Txi
}

type LocalGraphInfoProvider interface {
	GetMaxWeight() uint64
}
