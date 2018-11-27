package types

import (
	"math/big"
	"github.com/annchain/OG/types"
	"fmt"
)

type StateObject struct {
	Balance  *big.Int
	Nonce    uint64
	Code     []byte
	CodeHash types.Hash
	States   map[types.Hash]types.Hash
	Suicided bool
}

func (s *StateObject) String() string {
	return fmt.Sprintf("Balance %s Nonce %d CodeLen: %d CodeHash: %s", s.Balance, s.Nonce, len(s.Code), s.CodeHash.String())
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
		d.States[k] = v
	}

	d.Suicided = s.Suicided
	return d
}

func NewStateObject() *StateObject {
	return &StateObject{
		Balance: big.NewInt(0),
		States:  make(map[types.Hash]types.Hash),
	}
}
