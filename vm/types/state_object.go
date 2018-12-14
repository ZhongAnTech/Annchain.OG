package types

import (
	"fmt"
	"github.com/annchain/OG/types"
	"math/big"
)

type StateObject struct {
	Balance  *big.Int
	Nonce    uint64
	Code     []byte
	CodeHash types.Hash
	States   map[types.Hash]types.Hash
	Suicided bool
	Version  int
}

func (s *StateObject) String() string {
	return fmt.Sprintf("Balance %s Nonce %d CodeLen: %d CodeHash: %s States: %d Version: %d", s.Balance, s.Nonce, len(s.Code), s.CodeHash.String(), len(s.States), s.Version)
}

func (s *StateObject) Empty() bool {
	return s.Nonce == 0 && s.Balance.Sign() == 0 && s.CodeHash.Empty()
}
func (s *StateObject) Copy() (d *StateObject) {
	d = NewStateObject()
	d.CodeHash = s.CodeHash
	//d.CodeHash.SetBytes(s.CodeHash.Bytes[:])
	d.Code = s.Code
	//copy(d.Code, s.Code)
	d.Balance = s.Balance
	d.Nonce = s.Nonce

	d.States = make(map[types.Hash]types.Hash)
	for k, v := range s.States {
		d.States[types.BytesToHash(k.Bytes[:])] = types.BytesToHash(v.Bytes[:])
	}

	d.Suicided = s.Suicided
	d.Version = s.Version + 1
	return d
}

func NewStateObject() *StateObject {
	return &StateObject{
		Balance: big.NewInt(0),
		States:  make(map[types.Hash]types.Hash),
	}
}
