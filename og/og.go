package og

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/core"
	"github.com/sirupsen/logrus"
)

type Og struct {
	dag    *core.Dag
	txpool *core.TxPool

	accountManager *account.AccountManager

	manager *Manager
}

func (og *Og) Start() {
	logrus.Info("OG Started")
}
func (og *Og) Stop() {
	logrus.Info("OG Stopped")
}
