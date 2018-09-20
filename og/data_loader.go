package og

import (
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

type DataLoader struct {
	Dag    *core.Dag
	TxPool *core.TxPool
}

func (d *DataLoader) Start() {
	go d.LoadLocalData()
}

func (d *DataLoader) Stop() {
	logrus.Info("dataLoader received stop signal. Quiting...")
}

func (d *DataLoader) Name() string {
	return "DataLoader"
}

// LoadLocalData will load all necessary data (db, status, etc) from local database.
// If there is no data or data corrupted, rebuild.
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
	return &types.Sequencer{
		Issuer:            types.HexToAddress("0x00"),
		ContractHashOrder: []types.Hash{},
		Id:                0,
		TxBase: types.TxBase{
			Type:         types.TxBaseTypeSequencer,
			Hash:         types.HexToHash("0x00"),
			Height:       0,
			AccountNonce: 0,
		},
	}
}
func (loader *DataLoader) GenerateGenesisBalance() map[types.Address]*math.BigInt {
	return map[types.Address]*math.BigInt{}
}
