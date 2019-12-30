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
package og

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/types"
	core2 "github.com/annchain/OG/ogcore/ledger"
	"github.com/annchain/OG/ogcore/pool"

	"github.com/sirupsen/logrus"
)

type DataLoader struct {
	Dag    *core2.Dag
	TxPool *pool.TxPool
}

func (d *DataLoader) Start() {
	goroutine.New(d.LoadLocalData)
}

func (d *DataLoader) Stop() {
	logrus.Info("dataLoader received stop signal. Quiting...")
}

func (d *DataLoader) Name() string {
	return "DataLoader"
}

// LoadLocalData will load all necessary Data (db, status, etc) from local database.
// If there is no Data or Data corrupted, rebuild.
func (d *DataLoader) LoadLocalData() {
	genesis := d.Dag.Genesis()
	if genesis == nil {
		// write genesis and flush it to database
		genesis = d.GenerateGenesis()
		genesisBalance := d.GenerateGenesisBalance()
		d.Dag.Init(genesis, genesisBalance)
		// init tips
		d.TxPool.Init(d.Dag.LatestSequencer())
	}
}

func (d *DataLoader) GenerateGenesis() *types.Sequencer {
	from := common.HexToAddress("0x00")
	return &types.Sequencer{
		Issuer: &from,
		TxBase: types.TxBase{
			Type:         types.TxBaseTypeSequencer,
			Hash:         common.HexToHash("0x00"),
			Height:       0,
			AccountNonce: 0,
		},
	}
}
func (loader *DataLoader) GenerateGenesisBalance() map[common.Address]*math.BigInt {
	return map[common.Address]*math.BigInt{}
}
