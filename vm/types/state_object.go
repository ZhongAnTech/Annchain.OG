package types

import (
	"fmt"
	"github.com/annchain/OG/arefactor/og/types"
	"math/big"
)

type Storage map[types.Hash]types.Hash

type StateObject struct {
	Balance     *big.Int
	Nonce       uint64
	Code        []byte
	CodeHash    types.Hash
	Suicided    bool
	Version     int
	dirtyStates Storage
	DirtyCode   bool
	DirtySO     bool
}

func (s *StateObject) String() string {
	return fmt.Sprintf("Balance %s Nonce %d CodeLen: %d CodeHash: %s States: %d Version: %d", s.Balance, s.Nonce, len(s.Code), s.CodeHash.String(), len(s.dirtyStates), s.Version)
}

func (s *StateObject) Empty() bool {
	return s.DirtySO == false && len(s.dirtyStates) != 0
}
func (s *StateObject) Copy() (d *StateObject) {
	d = NewStateObject()
	d.Balance = s.Balance
	d.Nonce = s.Nonce
	d.Code = s.Code
	d.CodeHash = s.CodeHash
	d.Suicided = s.Suicided
	d.DirtySO = false
	d.dirtyStates = make(map[types.Hash]types.Hash)
	for k, v := range s.dirtyStates {
		d.dirtyStates[types.BytesToHash(k.Bytes[:])] = types.BytesToHash(v.Bytes[:])
	}

	d.Version = s.Version + 1
	return d
}

func NewStateObject() *StateObject {
	return &StateObject{
		Balance:     big.NewInt(0),
		dirtyStates: make(map[types.Hash]types.Hash),
	}
}

func NewStorage() Storage {
	a := make(map[types.Hash]types.Hash)
	return a
}
