package og

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/core"
)

type Og struct {
	dag    *core.Dag
	txpool *core.TxPool

	accountManager *account.AccountManager

	manager *Manager
}

func (og *Og) Start() {

}
