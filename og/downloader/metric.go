package downloader

import (
	"github.com/annchain/OG/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("eth/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("eth/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("eth/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("eth/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("eth/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("eth/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("eth/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("eth/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("eth/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("eth/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("eth/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("eth/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("eth/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("eth/downloader/states/drop", nil)
)
