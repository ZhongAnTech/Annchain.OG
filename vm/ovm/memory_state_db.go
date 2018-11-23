package ovm

import (
	"math/big"
	"github.com/annchain/OG/types"
	vmtypes "github.com/annchain/OG/vm/types"
	"github.com/annchain/OG/vm/eth/common"
	"github.com/annchain/OG/vm/eth/crypto"
	"strings"
	"fmt"
)

type stateObject struct {
	balance  *big.Int
	nonce    uint64
	code     []byte
	codeHash types.Hash
}

func (s *stateObject) String() string {
	return fmt.Sprintf("Balance %s Nonce %d CodeLen: %d CodeHash: %s", s.balance, s.nonce, len(s.code), s.codeHash.String())
}

func newStateObject() *stateObject {
	return &stateObject{
		balance: big.NewInt(0),
	}
}

type MemoryStateDB struct {
	ledger map[types.Address]*stateObject
}

func NewMemoryStateDB() *MemoryStateDB {
	return &MemoryStateDB{
		ledger: make(map[types.Address]*stateObject),
	}
}

func (m *MemoryStateDB) CreateAccount(addr types.Address) {
	m.ledger[addr] = newStateObject()
}

func (m *MemoryStateDB) SubBalance(addr types.Address, v *big.Int) {
	m.ledger[addr].balance = new(big.Int).Sub(m.ledger[addr].balance, v)
}

func (m *MemoryStateDB) AddBalance(addr types.Address, v *big.Int) {
	m.ledger[addr].balance = new(big.Int).Add(m.ledger[addr].balance, v)
}

func (m *MemoryStateDB) GetBalance(addr types.Address) *big.Int {
	if v, ok := m.ledger[addr]; ok {
		return v.balance
	}
	return common.Big0
}

func (m *MemoryStateDB) GetNonce(addr types.Address) uint64 {
	if v, ok := m.ledger[addr]; ok {
		return v.nonce
	}
	return 0
}

func (m *MemoryStateDB) SetNonce(addr types.Address, nonce uint64) {
	if v, ok := m.ledger[addr]; ok {
		v.nonce = nonce
	}
}

func (m *MemoryStateDB) GetCodeHash(addr types.Address) types.Hash {
	if v, ok := m.ledger[addr]; ok {
		return v.codeHash
	}
	return types.Hash{}
}

func (m *MemoryStateDB) GetCode(addr types.Address) []byte {
	if v, ok := m.ledger[addr]; ok {
		return v.code
	}
	return nil
}

func (m *MemoryStateDB) SetCode(addr types.Address, code []byte) {
	if v, ok := m.ledger[addr]; ok {
		v.code = code
		v.codeHash = crypto.Keccak256Hash(code)
	}
}

func (m *MemoryStateDB) GetCodeSize(addr types.Address) int {
	if v, ok := m.ledger[addr]; ok {
		return len(v.code)
	}
	return 0
}

func (m *MemoryStateDB) AddRefund(v uint64) {
	panic("implement me")
}

func (m *MemoryStateDB) SubRefund(v uint64) {
	panic("implement me")
}

func (m *MemoryStateDB) GetRefund() uint64 {
	panic("implement me")
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
	panic("implement me")
}

func (m *MemoryStateDB) AddPreimage(hash types.Hash, preImage []byte) {
	panic("implement me")
}

func (m *MemoryStateDB) ForEachStorage(addr types.Address, cb func(key, value types.Hash) bool) {
	panic("implement me")
}

func (m *MemoryStateDB) Dump() string {
	b := strings.Builder{}
	for k, v := range m.ledger {
		b.WriteString(fmt.Sprintf("%s: %s\n", k.String(), v))
	}
	return b.String()
}
