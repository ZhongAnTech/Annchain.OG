package ogcore

import (
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/model"
	"github.com/sirupsen/logrus"
)

type OgCore struct {
	statusData model.OgStatusData
	EventBus   EventBus
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

func (o *OgCore) HandleNewTx(tx *types.Tx) {

}

func (o *OgCore) HandleNewSequencer(seq *types.Sequencer) {

}

func (o *OgCore) LoadHeightTxs(height uint64, offset uint32) {

}