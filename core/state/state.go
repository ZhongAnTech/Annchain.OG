package state

import (
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
)

//go:generate msgp

//msgp:tuple State
type State struct {
	Address 	types.Address
	Balance		*math.BigInt
	Nonce		uint64
}
func NewState(addr types.Address) *State {
	s := &State{}
	s.Address = addr
	s.Balance = math.NewBigInt(0)
	s.Nonce = 0
	return s
}

func (s *State) AddBalance(increment *math.BigInt) {
	// check if increment is zero
	if increment.Sign() == 0 {
		return
	}
	s.SetBalance(s.Balance.Add(increment))
}

func (s *State) SubBalance(decrement *math.BigInt) {
	// check if decrement is zero
	if decrement.Sign() == 0 {
		return
	}
	s.SetBalance(s.Balance.Sub(decrement))
}

func (s *State) SetBalance(balance *math.BigInt) {
	s.Balance = balance
}

func (s *State) SetNonce(nonce uint64) {
	s.Nonce = nonce
}
