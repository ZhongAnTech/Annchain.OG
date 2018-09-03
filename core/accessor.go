package core

import (
	"bytes"
	"encoding/json"

	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common/math"
)

var (
	prefixTransactionKey = []byte("tk")
	prefixTransaction = []byte("tx")
	prefixSequencer = []byte("sq")

	prefixAddressBalanceKey = []byte("balance")
	// prefixContractState = []byte("con")
)

func transactionKey(hash types.Hash) []byte {
	return append(prefixTransactionKey, hash.ToBytes()...)
}

func addressBalanceKey(addr types.Address) []byte {
	return append(prefixAddressBalanceKey, addr.ToBytes()...)
}

// func contractStateKey(addr, contractAddr types.Address) []byte {
// 	return append(prefixContractState, append(addr.ToBytes(), contractAddr.ToBytes()...)...)
// }

type Accessor struct {
	db	ogdb.Database
}
func NewAccessor(db ogdb.Database) *Accessor {
	return &Accessor{ db: db }
}

// ReadTransaction get tx or sequencer from ogdb.
func (da *Accessor) ReadTransaction(hash types.Hash) types.Txi {
	data, _ := da.db.Get(transactionKey(hash))
	if len(data) == 0 {
		return nil
	}
	prefix := data[:2]
	if bytes.Equal(prefix, prefixTransaction) {
		var tx types.Tx
		// TODO use other decode function
		err := json.Unmarshal(data, &tx)
		if err != nil { return nil }
		return &tx
	} 
	if bytes.Equal(prefix, prefixSequencer) {
		var sq types.Sequencer
		// TODO use other decode function
		err := json.Unmarshal(data, &sq)
		if err != nil { return nil }
		return &sq
	} 
	return nil
}

// WriteTransaction write the tx or sequencer into ogdb, data will be overwritten
// if it already exist in db.
func (da *Accessor) WriteTransaction(txi types.Txi) error {
	key := transactionKey(txi.Hash())
	// TODO use other encode function
	data, err := json.Marshal(txi)
	if err != nil { 
		return err 
	}
	return da.db.Put(key, data)
}

// DeleteTransaction delete the tx or sequencer.
func (da *Accessor) DeleteTransaction(hash types.Hash) error {
	// TODO
	return nil
}

// ReadBalance get the balance of an address.
func (da *Accessor) ReadBalance(addr types.Address) *math.BigInt {
	// TODO
	return nil
}

// SetBalance write the balance of an address into ogdb.  
// Data will be overwritten if it already exist in db. 
func (da *Accessor) SetBalance(addr types.Address, value *math.BigInt) error {
	// TODO
	return nil
}

// DeleteBalance delete the balance of an address.
func (da *Accessor) DeleteBalance(addr types.Address) error {
	// TODO
	return nil
}

// AddBalance adds an amount of value to the address balance.
func (da *Accessor) AddBalance(addr types.Address, amount *math.BigInt) error {
	// TODO
	return nil
}

// SubBalance subs an amount of value to the address balance.
func (da *Accessor) SubBalance(addr types.Address, ammount *math.BigInt) error {
	// TODO
	return nil
}









