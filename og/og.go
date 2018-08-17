package og

import (
	"github.com/annchain/OG/account"
)

type Og struct {
	dag    *Dag
	txpool *TxPool

	accountManager *account.AccountManager

	manager *Manager
}

func (og *Og) Start() {}
