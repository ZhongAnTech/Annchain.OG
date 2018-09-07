package core

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
)

var (
	prefixGenesisKey	 = []byte("genesis")
	prefixLatestSeqKey	 = []byte("latestseq")

	prefixTransactionKey = []byte("tk")
	prefixTransaction    = []byte("tx")
	prefixSequencer      = []byte("sq")

	prefixAddressBalanceKey = []byte("ba")
	// prefixContractState = []byte("con")
)

func genesisKey() []byte {
	return prefixGenesisKey
}

func latestSequencerKey() []byte {
	return prefixLatestSeqKey
}

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
	db ogdb.Database
}

func NewAccessor(db ogdb.Database) *Accessor {
	return &Accessor{db: db}
}

// ReadGenesis get genesis sequencer from db. 
// return nil if there is no genesis.
func (da *Accessor) ReadGenesis() *types.Sequencer {
	data, _ := da.db.Get(genesisKey())
	if len(data) == 0 {
		return nil
	}
	// TODO use other decode function
	var seq types.Sequencer
	err := json.Unmarshal(data, &seq)
	if err != nil {
		return nil
	}
	return &seq
}

// WriteGenesis writes geneis into db.
func (da *Accessor) WriteGenesis(genesis *types.Sequencer) error {
	data, err := json.Marshal(genesis)
	if err != nil {
		return err
	}
	return da.db.Put(genesisKey(), data)
}

// ReadLatestSequencer get latest sequencer from db. 
// return nil if there is no sequencer.
func (da *Accessor) ReadLatestSequencer() *types.Sequencer {
	data, _ := da.db.Get(latestSequencerKey())
	if len(data) == 0 {
		return nil
	}
	// TODO use other decode function
	var seq types.Sequencer
	err := json.Unmarshal(data, &seq)
	if err != nil {
		return nil
	}
	return &seq
}

// WriteGenesis writes latest sequencer into db.
func (da *Accessor) WriteLatestSequencer(seq *types.Sequencer) error {
	data, err := json.Marshal(seq)
	if err != nil {
		return err
	}
	return da.db.Put(latestSequencerKey(), data)
}

// ReadTransaction get tx or sequencer from ogdb.
func (da *Accessor) ReadTransaction(hash types.Hash) types.Txi {
	data, _ := da.db.Get(transactionKey(hash))
	if len(data) == 0 {
		return nil
	}
	prefixLen := len(prefixTransaction)
	prefix := data[:prefixLen]
	data = data[prefixLen:]
	if bytes.Equal(prefix, prefixTransaction) {
		var tx types.Tx
		// TODO use other decode function
		err := json.Unmarshal(data, &tx)
		if err != nil {
			return nil
		}
		return &tx
	}
	if bytes.Equal(prefix, prefixSequencer) {
		var sq types.Sequencer
		// TODO use other decode function
		err := json.Unmarshal(data, &sq)
		if err != nil {
			return nil
		}
		return &sq
	}
	return nil
}

// WriteTransaction write the tx or sequencer into ogdb, data will be overwritten
// if it already exist in db.
func (da *Accessor) WriteTransaction(tx types.Txi) error {
	var prefix []byte
	switch tx.(type) {
	case *types.Tx:
		prefix = prefixTransaction
	case *types.Sequencer:
		prefix = prefixSequencer
	default:
		return fmt.Errorf("unknown tx type, must be *Tx or *Sequencer")
	}
	// TODO use other encode function
	data, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	data = append(prefix, data...)
	return da.db.Put(transactionKey(tx.MinedHash()), data)
}

// DeleteTransaction delete the tx or sequencer.
func (da *Accessor) DeleteTransaction(hash types.Hash) error {
	return da.db.Delete(transactionKey(hash))
}

// ReadBalance get the balance of an address.
func (da *Accessor) ReadBalance(addr types.Address) *math.BigInt {
	data, _ := da.db.Get(addressBalanceKey(addr))
	if len(data) == 0 {
		return math.NewBigInt(0)
	}
	var bigint math.BigInt
	// TODO use other encode function
	err := json.Unmarshal(data, &bigint)
	if err != nil {
		return nil
	}
	return &bigint
}

// SetBalance write the balance of an address into ogdb.
// Data will be overwritten if it already exist in db.
func (da *Accessor) SetBalance(addr types.Address, value *math.BigInt) error {
	if value.Value.Abs(value.Value).Cmp(value.Value) != 0 {
		return fmt.Errorf("the value of the balance must be positive!")
	}
	// TODO use other encode function
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return da.db.Put(addressBalanceKey(addr), data)
}

// DeleteBalance delete the balance of an address.
func (da *Accessor) DeleteBalance(addr types.Address) error {
	return da.db.Delete(addressBalanceKey(addr))
}

// AddBalance adds an amount of value to the address balance. Note that AddBalance
// doesn't hold any locks so upper level program must manage this.
func (da *Accessor) AddBalance(addr types.Address, amount *math.BigInt) error {
	if amount.Value.Abs(amount.Value).Cmp(amount.Value) != 0 {
		return fmt.Errorf("add amount must be positive!")
	}
	balance := da.ReadBalance(addr)
	// no balance exists
	if balance == nil {
		return da.SetBalance(addr, amount)
	}
	newBalanceValue := balance.Value.Add(balance.Value, amount.Value)
	return da.SetBalance(addr, &math.BigInt{Value: newBalanceValue})
}

// SubBalance subs an amount of value to the address balance. Note that SubBalance
// doesn't hold any locks so upper level program must manage this.
func (da *Accessor) SubBalance(addr types.Address, amount *math.BigInt) error {
	if amount.Value.Abs(amount.Value).Cmp(amount.Value) != 0 {
		return fmt.Errorf("add amount must be positive!")
	}
	balance := da.ReadBalance(addr)
	// no balance exists
	if balance == nil {
		return fmt.Errorf("address %s has no balance yet, cannot process sub", addr.String())
	}
	if balance.Value.Cmp(amount.Value) == -1 {
		return fmt.Errorf("address %s has no enough balance to sub. balance: %d, sub amount: %d",
			addr.String(), balance.GetInt64(), amount.GetInt64())
	}
	newBalanceValue := balance.Value.Sub(balance.Value, amount.Value)
	return da.SetBalance(addr, &math.BigInt{Value: newBalanceValue})
}
