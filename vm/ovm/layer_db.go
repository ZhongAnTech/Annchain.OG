package ovm

import (
	"github.com/annchain/OG/types"
	vmtypes "github.com/annchain/OG/vm/types"

	"fmt"
	"math/big"
	"strings"

	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/pkg/errors"
)

var MAX_LAYER = 1024
var FAST_FAIL = false
var empty = types.Hash{}

// LayerStateDB is the cascading storage for contracts.
// It consists of multiple layers, each of which represents the result of a contract.
// e.g.
// -.-#-+-#-----	<- Latest changes after contract
// --+-------+--	<- ...
// ----.----.---	<- first contract
// -------------	<- StateDB (on disk)
// When accessing, watch from the top to the bottom.
// Same mechanism as Docker image layers.
// Add a layer each time a new contract is run
type LayerStateDB struct {
	Layers      []vmtypes.StateDBDebug
	activeLayer vmtypes.StateDBDebug
}

func (l *LayerStateDB) GetStateObject(addr types.Address) *vmtypes.StateObject {
	for i := len(l.Layers) - 1; i >= 0; i-- {
		layer := l.Layers[i]
		if so := layer.GetStateObject(addr); so != nil {
			// return a copy in case you need to modify it.
			if i == len(l.Layers)-1 {
				return so
			} else {
				return so.Copy()
			}

		}
	}
	return nil
}

func (l *LayerStateDB) SetStateObject(addr types.Address, stateObject *vmtypes.StateObject) {
	l.activeLayer.SetStateObject(addr, stateObject)
	stateObject.DirtySO = true
}

func NewLayerDB(baseLayer vmtypes.StateDBDebug) *LayerStateDB {
	return &LayerStateDB{
		Layers:      []vmtypes.StateDBDebug{baseLayer},
		activeLayer: baseLayer,
	}
}

func (l *LayerStateDB) NewLayer() (index int, err error) {
	// add a new memory layer onto the current layer stack
	index = len(l.Layers)
	if index > MAX_LAYER {
		err = errors.New("max layer count reached")
		return
	}
	l.activeLayer = NewMemoryStateDB()
	l.Layers = append(l.Layers, l.activeLayer)
	return
}

func (l *LayerStateDB) PopLayer(index int) (err error) {
	if len(l.Layers) != index+1 {
		return errors.New("only top layer can be removed")
	}
	if len(l.Layers) == 1 {
		return errors.New("no more layers can be removed")
	}
	l.Layers = l.Layers[:len(l.Layers)-1]
	l.activeLayer = l.Layers[len(l.Layers)-1]
	return
}

func (l *LayerStateDB) String() string {
	buffer := strings.Builder{}
	for i, layer := range l.Layers {
		if i == 0 {
			buffer.WriteString(fmt.Sprintf("Layer BOTTOM\r\n"))
		} else {
			buffer.WriteString(fmt.Sprintf("Layer %d\r\n", i))
		}

		buffer.WriteString(layer.String())
		buffer.WriteString("\r\n")
	}
	return buffer.String()
}

func (l *LayerStateDB) CreateAccount(addr types.Address) {
	if so := l.GetStateObject(addr); so == nil {
		l.activeLayer.CreateAccount(addr)
	} else {
		if FAST_FAIL {
			panic("address already exists")
		}
	}
}

func (l *LayerStateDB) SubBalance(addr types.Address, value *math.BigInt) {
	if so := l.GetStateObject(addr); so != nil {
		so.Balance = new(big.Int).Sub(so.Balance, value.Value)
		// store to this layer
		l.SetStateObject(addr, so)
	} else {
		if FAST_FAIL {
			panic("address not exists")
		}
	}
}

func (l *LayerStateDB) AddBalance(addr types.Address, value *math.BigInt) {
	if so := l.GetStateObject(addr); so != nil {
		so.Balance = new(big.Int).Add(so.Balance, value.Value)
		// store to this layer
		l.SetStateObject(addr, so)
	} else {
		if FAST_FAIL {
			panic("address not exists")
		}
	}
}

func (l *LayerStateDB) GetBalance(addr types.Address) *math.BigInt {
	if so := l.GetStateObject(addr); so != nil {
		return math.NewBigIntFromBigInt(so.Balance)
	} else {
		if FAST_FAIL {
			panic("address not exists")
		}
	}
	return math.NewBigIntFromBigInt(common.Big0)
}

func (l *LayerStateDB) GetNonce(addr types.Address) uint64 {
	if so := l.GetStateObject(addr); so != nil {
		return so.Nonce
	} else {
		if FAST_FAIL {
			panic("address not exists")
		}
	}
	return 0
}

func (l *LayerStateDB) SetNonce(addr types.Address, nonce uint64) {
	if so := l.GetStateObject(addr); so != nil {
		so.Nonce = nonce
		l.SetStateObject(addr, so)
	} else {
		if FAST_FAIL {
			panic("address not exists")
		}
	}
}

func (l *LayerStateDB) GetCodeHash(addr types.Address) types.Hash {
	if so := l.GetStateObject(addr); so != nil {
		return so.CodeHash
	} else {
		if FAST_FAIL {
			panic("address not exists")
		}
	}
	return types.Hash{}
}

func (l *LayerStateDB) GetCode(addr types.Address) []byte {
	if so := l.GetStateObject(addr); so != nil {
		return so.Code
	} else {
		if FAST_FAIL {
			panic("address not exists")
		}
	}
	return nil
}

func (l *LayerStateDB) SetCode(addr types.Address, code []byte) {
	if so := l.GetStateObject(addr); so != nil {
		so.Code = code
		so.CodeHash = crypto.Keccak256Hash(code)
		so.DirtyCode = true
		l.SetStateObject(addr, so)
	} else {
		if FAST_FAIL {
			panic("address not exists")
		}
	}
}

func (l *LayerStateDB) GetCodeSize(addr types.Address) int {
	so := l.GetStateObject(addr)
	if so == nil {
		return 0
	}
	if so.Code != nil {
		return len(so.Code)
	}
	return 0
}

func (l *LayerStateDB) AddRefund(value uint64) {
	l.activeLayer.AddRefund(value)
}

func (l *LayerStateDB) SubRefund(value uint64) {
	l.activeLayer.SubRefund(value)
}

func (l *LayerStateDB) GetRefund() uint64 {
	sum := uint64(0)
	for _, layer := range l.Layers {
		sum += layer.GetRefund()
	}
	return sum
}

func (l *LayerStateDB) GetCommittedState(addr types.Address, hash types.Hash) types.Hash {
	// TODO: committed state
	panic("implement me")
}

func (l *LayerStateDB) GetState(addr types.Address, key types.Hash) types.Hash {
	for i := len(l.Layers) - 1; i >= 0; i-- {
		layer := l.Layers[i]
		if so := layer.GetState(addr, key); so != empty {
			return so
		}
	}
	return types.Hash{}
}

func (l *LayerStateDB) SetState(addr types.Address, key types.Hash, value types.Hash) {
	l.activeLayer.SetState(addr, key, value)
}

func (l *LayerStateDB) Suicide(addr types.Address) bool {
	so := l.GetStateObject(addr)
	if so == nil {
		return false
	}
	so.Suicided = true
	so.Balance = new(big.Int)
	l.SetStateObject(addr, so)
	return true
}

func (l *LayerStateDB) HasSuicided(addr types.Address) bool {
	so := l.GetStateObject(addr)
	if so == nil {
		return false
	}
	return so.Suicided
}

func (l *LayerStateDB) Exist(addr types.Address) bool {
	so := l.GetStateObject(addr)
	return so != nil
}

func (l *LayerStateDB) Empty(addr types.Address) bool {
	so := l.GetStateObject(addr)
	return so == nil || so.Empty()
}

func (l *LayerStateDB) RevertToSnapshot(i int) {
	l.Layers = l.Layers[0:i]
}

func (l *LayerStateDB) Snapshot() int {
	l.NewLayer()
	return len(l.Layers) - 1
}

func (l *LayerStateDB) AddLog(log *vmtypes.Log) {
	l.activeLayer.AddLog(log)
}

func (l *LayerStateDB) AddPreimage(hash types.Hash, code []byte) {
	l.activeLayer.AddPreimage(hash, code)
}

func (l *LayerStateDB) ForEachStorage(addr types.Address, f func(types.Hash, types.Hash) bool) {
	// TODO: foreach storage
	panic("implement me")
}

// MergeChanges merges all layers that are above the bottom layer to one layer
func (l *LayerStateDB) MergeChanges() {
	// if there is only bottom layer, ignore
	if len(l.Layers) <= 2 {
		return
	}
	// put all changes to layer #1

	for i := 2; i < len(l.Layers); i++ {
		l.mergeLayer(1, i)
	}
	// delete all layers after #1
	l.Layers = l.Layers[0:2]
}
func (l *LayerStateDB) mergeLayer(toLayerIndex int, fromLayerIndex int) {
	toLayer, toOk := l.Layers[toLayerIndex].(*MemoryStateDB)
	fromLayer, fromOk := l.Layers[fromLayerIndex].(*MemoryStateDB)

	if !toOk {
		panic("to layer does not support merging")
	}
	if !fromOk {
		panic("from layer does not support merging")
	}

	for k, v := range fromLayer.soLedger {
		toLayer.soLedger[k] = v
	}
	for k, v := range fromLayer.kvLedger {
		toLayer.kvLedger[k] = v
	}

}

func (l *LayerStateDB) CurrentLayer() int {
	return len(l.Layers) - 1
}
