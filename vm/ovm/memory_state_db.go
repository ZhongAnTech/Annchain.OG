package ovm

import (
	"math/big"
	"github.com/annchain/OG/types"
	vmtypes "github.com/annchain/OG/vm/types"
)

type MemoryStateDB struct{
	
}

func (MemoryStateDB) CreateAccount(types.Address) {
	panic("implement me")
}

func (MemoryStateDB) SubBalance(types.Address, *big.Int) {
	panic("implement me")
}

func (MemoryStateDB) AddBalance(types.Address, *big.Int) {
	panic("implement me")
}

func (MemoryStateDB) GetBalance(types.Address) *big.Int {
	panic("implement me")
}

func (MemoryStateDB) GetNonce(types.Address) uint64 {
	panic("implement me")
}

func (MemoryStateDB) SetNonce(types.Address, uint64) {
	panic("implement me")
}

func (MemoryStateDB) GetCodeHash(types.Address) types.Hash {
	panic("implement me")
}

func (MemoryStateDB) GetCode(types.Address) []byte {
	panic("implement me")
}

func (MemoryStateDB) SetCode(types.Address, []byte) {
	panic("implement me")
}

func (MemoryStateDB) GetCodeSize(types.Address) int {
	panic("implement me")
}

func (MemoryStateDB) AddRefund(uint64) {
	panic("implement me")
}

func (MemoryStateDB) SubRefund(uint64) {
	panic("implement me")
}

func (MemoryStateDB) GetRefund() uint64 {
	panic("implement me")
}

func (MemoryStateDB) GetCommittedState(types.Address, types.Hash) types.Hash {
	panic("implement me")
}

func (MemoryStateDB) GetState(types.Address, types.Hash) types.Hash {
	panic("implement me")
}

func (MemoryStateDB) SetState(types.Address, types.Hash, types.Hash) {
	panic("implement me")
}

func (MemoryStateDB) Suicide(types.Address) bool {
	panic("implement me")
}

func (MemoryStateDB) HasSuicided(types.Address) bool {
	panic("implement me")
}

func (MemoryStateDB) Exist(types.Address) bool {
	panic("implement me")
}

func (MemoryStateDB) Empty(types.Address) bool {
	panic("implement me")
}

func (MemoryStateDB) RevertToSnapshot(int) {
	panic("implement me")
}

func (MemoryStateDB) Snapshot() int {
	panic("implement me")
}

func (MemoryStateDB) AddLog(*vmtypes.Log) {
	panic("implement me")
}

func (MemoryStateDB) AddPreimage(types.Hash, []byte) {
	panic("implement me")
}

func (MemoryStateDB) ForEachStorage(types.Address, func(types.Hash, types.Hash) bool) {
	panic("implement me")
}
