package ovm

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/eth/common"
	vmtypes "github.com/annchain/OG/vm/types"
	"math/big"
	"sort"
	"strings"
)

type MemoryStateDB struct {
	ledger map[types.Address]*vmtypes.StateObject
	refund uint64
}

func (m *MemoryStateDB) GetStateObject(addr types.Address) *vmtypes.StateObject {
	if v, ok := m.ledger[addr]; ok {
		return v
	}
	return nil
}

func (m *MemoryStateDB) SetStateObject(addr types.Address, stateObject *vmtypes.StateObject) {
	m.ledger[addr] = stateObject
}

func NewMemoryStateDB() *MemoryStateDB {
	return &MemoryStateDB{
		ledger: make(map[types.Address]*vmtypes.StateObject),
	}
}

func (m *MemoryStateDB) CreateAccount(addr types.Address) {
	if _, ok := m.ledger[addr]; ok {
		return
	}
	m.ledger[addr] = vmtypes.NewStateObject()
}

func (m *MemoryStateDB) SubBalance(addr types.Address, v *big.Int) {
	m.ledger[addr].Balance = new(big.Int).Sub(m.ledger[addr].Balance, v)
}

func (m *MemoryStateDB) AddBalance(addr types.Address, v *big.Int) {
	m.ledger[addr].Balance = new(big.Int).Add(m.ledger[addr].Balance, v)
}

func (m *MemoryStateDB) GetBalance(addr types.Address) *big.Int {
	if v, ok := m.ledger[addr]; ok {
		return v.Balance
	}
	return common.Big0
}

func (m *MemoryStateDB) GetNonce(addr types.Address) uint64 {
	if v, ok := m.ledger[addr]; ok {
		return v.Nonce
	}
	return 0
}

func (m *MemoryStateDB) SetNonce(addr types.Address, nonce uint64) {
	if v, ok := m.ledger[addr]; ok {
		v.Nonce = nonce
	}
}

func (m *MemoryStateDB) GetCodeHash(addr types.Address) types.Hash {
	if v, ok := m.ledger[addr]; ok {
		return v.CodeHash
	}
	return types.Hash{}
}

func (m *MemoryStateDB) GetCode(addr types.Address) []byte {
	if v, ok := m.ledger[addr]; ok {
		return v.Code
	}
	return nil
}

func (m *MemoryStateDB) SetCode(addr types.Address, code []byte) {
	if v, ok := m.ledger[addr]; ok {
		v.Code = code
		v.CodeHash = crypto.Keccak256Hash(code)
	}
}

func (m *MemoryStateDB) GetCodeSize(addr types.Address) int {
	if v, ok := m.ledger[addr]; ok {
		return len(v.Code)
	}
	return 0
}

func (m *MemoryStateDB) AddRefund(v uint64) {
	m.refund += v
}

func (m *MemoryStateDB) SubRefund(v uint64) {
	if v > m.refund {
		panic("Refund counter below zero")
	}
	m.refund -= v
}

func (m *MemoryStateDB) GetRefund() uint64 {
	return m.refund
}

func (m *MemoryStateDB) GetCommittedState(addr types.Address, hash types.Hash) types.Hash {
	panic("implement me")
}

func (m *MemoryStateDB) GetState(addr types.Address, key types.Hash) types.Hash {
	panic("implement me")
}

func (m *MemoryStateDB) SetState(addr types.Address, key types.Hash, value types.Hash) {
	panic("implement me")
}

func (m *MemoryStateDB) Suicide(addr types.Address) bool {
	panic("implement me")
}

func (m *MemoryStateDB) HasSuicided(addr types.Address) bool {
	panic("implement me")
}

func (m *MemoryStateDB) Exist(addr types.Address) bool {
	_, ok := m.ledger[addr]
	return ok
}

func (m *MemoryStateDB) Empty(addr types.Address) bool {
	panic("implement me")
}

func (m *MemoryStateDB) RevertToSnapshot(int) {
	panic("implement me")
}

func (m *MemoryStateDB) Snapshot() int {
	return 0
}

func (m *MemoryStateDB) AddLog(*vmtypes.Log) {
	// Maybe we don't care about logs sinces we have layerdb.
	return
}

func (m *MemoryStateDB) AddPreimage(hash types.Hash, preImage []byte) {
	// Any usage?
	return
}

func (m *MemoryStateDB) ForEachStorage(addr types.Address, cb func(key, value types.Hash) bool) {
	panic("implement me")
}

func (m *MemoryStateDB) String() string {
	b := strings.Builder{}
	for k, v := range m.ledger {
		b.WriteString(fmt.Sprintf("%s: %s\n", k.String(), v))
		if len(v.States) > 0 {
			var keys types.Hashes
			for sk := range v.States {
				keys = append(keys, sk)
			}
			sort.Sort(keys)

			for _, key := range keys {
				b.WriteString(fmt.Sprintf("-->  %s: %s\n", key.Hex(), v.States[key].Hex()))
			}
		}
	}
	return b.String()
}
