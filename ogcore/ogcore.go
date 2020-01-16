package ogcore

import (
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/interfaces"
	"github.com/annchain/OG/ogcore/model"
	"github.com/annchain/OG/ogcore/pool"
	"github.com/sirupsen/logrus"
)

// OgCore includes the basic components needed to build an Og
// Focus on internal affairs. Any communication modules should be in OgPartner.
// This means that a solo node will require OgCore ONLY.
type OgCore struct {
	statusData       model.OgStatusData
	EventBus         eventbus.EventBus
	LedgerTxProvider interfaces.LedgerTxProvider
	TxBuffer         *pool.TxBuffer
	TxPool           *pool.TxPool
}

func (o *OgCore) Name() string {
	return "OgCore"
}

func (o *OgCore) FireEvent(event eventbus.Event) {
	if o.EventBus != nil {
		o.EventBus.Route(event)
	}
}

func (o *OgCore) GetCurrentOgStatus() model.OgStatusData {
	return o.statusData
}

func (o *OgCore) HandleStatusData(status model.OgStatusData) {
	// compare the status with the current one.
	if !o.statusData.IsCompatible(status) {
		logrus.WithField("mine", o.statusData).WithField("theirs", status).Warn("StatusData not matched")
		return
	}
	if o.statusData.IsHeightNotLowerThan(status) {
		logrus.WithField("mine", o.statusData).WithField("theirs", status).Trace("we are not behind")
		return
	}
	// we are behind. start sync.
	o.FireEvent(&events.HeightBehindEvent{LatestKnownHeight: status.CurrentHeight})
}

func (o *OgCore) HandleEvent(ev eventbus.Event) {
	switch ev.GetEventType() {
	default:
		logrus.Warn("event type not supported by ogcore")
	}
}

func (o *OgCore) HandlerDescription(ev eventbus.EventType) string {
	switch ev {
	default:
		return "N/A"
	}
}

func (o *OgCore) LoadHeightTxs(height uint64, offset uint32, count uint32) []types.Txi {
	return o.LedgerTxProvider.GetHeightTxs(height, offset, count)
}
