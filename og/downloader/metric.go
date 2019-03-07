// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
