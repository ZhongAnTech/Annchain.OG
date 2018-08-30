package og

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/annchain/OG/account"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/ogdb"
)

type Og struct {
	Dag            *core.Dag
	Txpool         *core.TxPool
	AccountManager *account.AccountManager
	Manager        *Manager
}

func NewOg() (*Og, error) {
	og := &Og{}

	var (
		dagconfig    core.DagConfig
		txpoolconfig core.TxPoolConfig
	)

	db, derr := CreateDB()
	if derr != nil {
		return nil, derr
	}
	if err := viper.UnmarshalKey("dag", &dagconfig); err != nil {
		return nil, err
	}
	og.Dag = core.NewDag(dagconfig, db)

	if err := viper.UnmarshalKey("txpool", &txpoolconfig); err != nil {
		return nil, err
	}
	og.Txpool = core.NewTxPool(txpoolconfig, og.Dag)

	// TODO
	// account manager and protocol manager

	return og, nil
}

func (og *Og) Start() {

	logrus.Info("OG Started")
}
func (og *Og) Stop() {
	logrus.Info("OG Stopped")
}

func (og *Og) Name() string {
	return "OG"
}

func CreateDB() (ogdb.Database, error) {
	switch viper.GetString("db.name") {
	case "leveldb":
		path := viper.GetString("leveldb.path")
		cache := viper.GetInt("leveldb.cache")
		handles := viper.GetInt("leveldb.handles")
		return ogdb.NewLevelDB(path, cache, handles)
	default:
		return ogdb.NewMemDatabase(), nil
	}
}
