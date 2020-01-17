package ogcore

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/debug/debuglog"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/interfaces"
	"github.com/annchain/OG/ogcore/model"
	"github.com/annchain/OG/ogcore/pool"
)

// OgCore includes the basic components needed to build an Og
// Focus on internal affairs. Any communication modules should be in OgPartner.
// This means that a solo node will require OgCore ONLY.
type OgCore struct {
	debuglog.NodeLogger
	statusData       model.OgStatusData
	EventBus         eventbus.EventBus
	LedgerTxProvider interfaces.LedgerTxProvider
	TxBuffer         *pool.TxBuffer
	TxPool           *pool.TxPool
	KnownTxiCache    *pool.KnownTxiCache
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
		o.Logger.WithField("mine", o.statusData).WithField("theirs", status).Warn("StatusData not matched")
		return
	}
	if o.statusData.IsHeightNotLowerThan(status) {
		o.Logger.WithField("mine", o.statusData).WithField("theirs", status).Trace("we are not behind")
		return
	}
	// we are behind. start sync.
	o.FireEvent(&events.HeightBehindEvent{LatestKnownHeight: status.CurrentHeight})
}

func (o *OgCore) HandleEvent(ev eventbus.Event) {
	switch ev.GetEventType() {
	// duplicate event NewTxReceivedInPoolEventType
	//case events.NewTxReceivedInPoolEventType:
	//	evt := ev.(*events.NewTxReceivedInPoolEvent)
	//	// put into knownTxiCache
	//	o.KnownTxiCache.Put(evt.Tx)
	case events.NewTxLocallyGeneratedEventType:
		evt := ev.(*events.NewTxLocallyGeneratedEvent)
		// put into knownTxiCache
		o.KnownTxiCache.Put(evt.Tx)
	case events.NewSequencerLocallyGeneratedEventType:
		evt := ev.(*events.NewSequencerLocallyGeneratedEvent)
		// put into knownTxiCache
		o.KnownTxiCache.Put(evt.Sequencer)
	case events.TxReceivedEventType:
		evt := ev.(*events.TxReceivedEvent)
		// put into knownTxiCache
		o.KnownTxiCache.Put(evt.Tx)
	case events.SequencerReceivedEventType:
		evt := ev.(*events.SequencerReceivedEvent)
		// put into knownTxiCache
		o.KnownTxiCache.Put(evt.Sequencer)
	default:
		o.Logger.Warn("event type not supported by ogcore")
	}
}

func (o *OgCore) HandlerDescription(ev eventbus.EventType) string {
	switch ev {
	case events.NewTxLocallyGeneratedEventType:
		return "AddLocalTxToKnownCache"
	case events.NewSequencerLocallyGeneratedEventType:
		return "AddLocalSequencerToKnownCache"
	case events.TxReceivedEventType:
		return "AddRemoteTxToKnownCache"
	case events.SequencerReceivedEventType:
		return "AddRemoteSequencerToKnownCache"

	default:
		return "N/A"
	}
}

func (o *OgCore) LoadHeightTxis(height uint64, offset int, count int) []types.Txi {
	return o.LedgerTxProvider.GetHeightTxs(height, offset, count)
}

func (o *OgCore) LoadTxis(hashes common.Hashes, maxCount int) types.Txis {
	var txis types.Txis
	for i, hash := range hashes {
		txi, err := o.KnownTxiCache.Get(hash)
		if err != nil {
			continue
		}
		o.Logger.WithField("txi", txi).Trace("knownTxiCache provided txi")
		txis = append(txis, txi)
		i += 1
		if i >= maxCount {
			break
		}
	}
	return txis
}
