package core

import (
	"time"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common/math"
)

type StateDB struct {
	states 			map[types.Address]*AccountState
}

func (sd *StateDB) Load() {
	
}

func (sd *StateDB) Confirm() {

}

type AccountState struct {
	Address 	types.Address
	Nonce		uint64
	Balance		*math.BigInt
	LastBeat	time.Time
	IsDirty		bool
}







