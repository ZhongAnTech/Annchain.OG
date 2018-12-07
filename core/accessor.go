package core

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
)

var (
	prefixGenesisKey   = []byte("genesis")
	prefixLatestSeqKey = []byte("latestseq")

	prefixTransactionKey     = []byte("tx")
	prefixTxHashFlowKey      = []byte("fl")
	contentPrefixTransaction = []byte("cptx")
	contentPrefixSequencer   = []byte("cpsq")

	prefixTxSeqRelationKey = []byte("tsr")

	prefixAddrLatestNonceKey = []byte("aln")

	prefixSeqIdKey   = []byte("si")
	prefixTxIndexKey = []byte("ti")

	prefixAddressBalanceKey = []byte("ba")

	prefixStateKey = []byte("st")
	prefixConfimtime   = []byte("cf")
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

func confirmTimeKey(seqId uint64) []byte {
	suffix := prefixConfimtime
	return append(prefixConfimtime, append([]byte(strconv.FormatUint(seqId, 10)), suffix...)...)
}

func txHashFlowKey(addr types.Address, nonce uint64) []byte {
	keybody := append(addr.ToBytes(), []byte(strconv.FormatUint(nonce, 10))...)
	return append(prefixTxHashFlowKey, keybody...)
}

func txSeqRelationKey(hash types.Hash) []byte {
	return append(prefixTxSeqRelationKey, hash.ToBytes()...)
}

func addrLatestNonceKey(addr types.Address) []byte {
	return append(prefixAddrLatestNonceKey, addr.ToBytes()...)
}

func addressBalanceKey(addr types.Address) []byte {
	return append(prefixAddressBalanceKey, addr.ToBytes()...)
}

func seqIdKey(seqid uint64) []byte {
	suffix := prefixSeqIdKey
	return append(prefixSeqIdKey, append([]byte(strconv.FormatUint(seqid, 10)), suffix...)...)
}

func txIndexKey(seqid uint64) []byte {
	suffix := prefixTxIndexKey
	return append(prefixTxIndexKey, append([]byte(strconv.FormatUint(seqid, 10)), suffix...)...)
}

func stateKey(addr types.Address) []byte {
	return append(prefixStateKey, addr.ToBytes()...)
}

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
	var seq types.Sequencer
	_, err := seq.UnmarshalMsg(data)
	if err != nil {
		return nil
	}
	return &seq
}

// WriteGenesis writes geneis into db.
func (da *Accessor) WriteGenesis(genesis *types.Sequencer) error {
	data, err := genesis.MarshalMsg(nil)
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
	var seq types.Sequencer
	_, err := seq.UnmarshalMsg(data)
	if err != nil {
		return nil
	}
	return &seq
}

// WriteGenesis writes latest sequencer into db.
func (da *Accessor) WriteLatestSequencer(seq *types.Sequencer) error {
	data, err := seq.MarshalMsg(nil)
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
	prefixLen := len(contentPrefixTransaction)
	prefix := data[:prefixLen]
	data = data[prefixLen:]
	if bytes.Equal(prefix, contentPrefixTransaction) {
		var tx types.Tx
		tx.UnmarshalMsg(data)
		return &tx
	}
	if bytes.Equal(prefix, contentPrefixSequencer) {
		var sq types.Sequencer
		sq.UnmarshalMsg(data)
		return &sq
	}
	return nil
}

// ReadTxByNonce get tx from db by sender's address and nonce.
func (da *Accessor) ReadTxByNonce(addr types.Address, nonce uint64) types.Txi {
	data, _ := da.db.Get(txHashFlowKey(addr, nonce))
	if len(data) == 0 {
		return nil
	}
	hash := types.BytesToHash(data)
	return da.ReadTransaction(hash)
}

// WriteTxHashByNonce writes tx hash into db and construct key with address and nonce.
func (da *Accessor) WriteTxHashByNonce(addr types.Address, nonce uint64, hash types.Hash) error {
	data := hash.ToBytes()
	err := da.db.Put(txHashFlowKey(addr, nonce), data)
	if err != nil {
		return fmt.Errorf("write tx hash flow to db err: %v", err)
	}
	return nil
}

// ReadAddrLatestNonce get latest nonce of an address
func (da *Accessor) ReadAddrLatestNonce(addr types.Address) (uint64, error) {
	has, _ := da.HasAddrLatestNonce(addr)
	if !has {
		return 0, fmt.Errorf("not exists")
	}
	data, err := da.db.Get(addrLatestNonceKey(addr))
	if len(data) == 0 {
		return 0, err
	}
	nonce, parseErr := strconv.ParseUint(string(data), 10, 64)
	if parseErr != nil {
		return 0, fmt.Errorf("parse nonce from bytes to uint64 error: %v", parseErr)
	}
	return nonce, nil
}

// HasAddrLatestNonce returns true if addr already sent some txs.
func (da *Accessor) HasAddrLatestNonce(addr types.Address) (bool, error) {
	return da.db.Has(addrLatestNonceKey(addr))
}

func (da*Accessor)writeConfirmTime(cf *types.ConfirmTime) error {
	data, err := cf.MarshalMsg(nil)
	if err !=nil {
		return err
	}
	err = da.db.Put(confirmTimeKey(cf.SeqId), data)
	if err != nil {
		return fmt.Errorf("write tx to db batch err: %v", err)
	}

	return nil
}

func (da*Accessor)readConfirmTime(seqId uint64) *types.ConfirmTime {
	data, _ := da.db.Get(confirmTimeKey(seqId))
	if len(data) == 0 {
		return nil
	}
	var cf types.ConfirmTime
		_,err:=  cf.UnmarshalMsg(data)
		if err!=nil || cf.SeqId != seqId{
			return nil
		}
	return &cf
}

// WriteTransaction write the tx or sequencer into ogdb.
func (da *Accessor) WriteTransaction(putter ogdb.Putter, tx types.Txi) error {
	var prefix, data []byte
	var err error

	// write tx
	switch tx := tx.(type) {
	case *types.Tx:
		prefix = contentPrefixTransaction
		data, err = tx.MarshalMsg(nil)
	case *types.Sequencer:
		prefix = contentPrefixSequencer
		data, err = tx.MarshalMsg(nil)
	default:
		return fmt.Errorf("unknown tx type, must be *Tx or *Sequencer")
	}
	if err != nil {
		return fmt.Errorf("marshal tx %s err: %v", tx.GetTxHash().String(), err)
	}
	data = append(prefix, data...)
	err = da.db.Put(transactionKey(tx.GetTxHash()), data)
	if err != nil {
		return fmt.Errorf("write tx to db batch err: %v", err)
	}

	return nil
}

// ReadTxSeqRelation get the bound seq id of a tx
func (da *Accessor) ReadTxSeqRelation(hash types.Hash) (uint64, error) {
	data, err := da.db.Get(txSeqRelationKey(hash))
	if err != nil {
		return 0, err
	}
	seqid, errToUint := strconv.ParseUint(string(data), 10, 64)
	if errToUint != nil {
		return 0, errToUint
	}
	return seqid, nil
}

// WriteTxSeqRelation bind the seq id to a tx hash
func (da *Accessor) WriteTxSeqRelation(hash types.Hash, seqid uint64) error {
	data := []byte(strconv.FormatUint(seqid, 10))
	return da.db.Put(txSeqRelationKey(hash), data)
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
	_, err := bigint.UnmarshalMsg(data)
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
	data, err := value.MarshalMsg(nil)
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

// ReadSequencerById get sequencer from db by sequencer id.
func (da *Accessor) ReadSequencerById(seqid uint64) (*types.Sequencer, error) {
	data, _ := da.db.Get(seqIdKey(seqid))
	if len(data) == 0 {
		return nil, fmt.Errorf("sequencer with seqid %d not found", seqid)
	}
	var seq types.Sequencer
	_, err := seq.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}
	return &seq, nil
}

// WriteSequencerById stores the sequencer into db and indexed by its id.
func (da *Accessor) WriteSequencerById(seq *types.Sequencer) error {
	data, err := seq.MarshalMsg(nil)
	if err != nil {
		return err
	}
	return da.db.Put(seqIdKey(seq.Id), data)
}

// ReadIndexedTxHashs get a list of txs that is confirmed by the sequencer that
// holds the id 'seqid'.
func (da *Accessor) ReadIndexedTxHashs(seqid uint64) (*types.Hashs, error) {
	data, _ := da.db.Get(txIndexKey(seqid))
	if len(data) == 0 {
		return nil, fmt.Errorf("tx hashs with seq id %d not found", seqid)
	}
	var hashs types.Hashs
	_, err := hashs.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}
	return &hashs, nil
}

// WriteIndexedTxHashs stores a list of tx hashs. These related hashs are all
// confirmed by sequencer that holds the id 'seqid'.
func (da *Accessor) WriteIndexedTxHashs(seqid uint64, hashs *types.Hashs) error {
	data, err := hashs.MarshalMsg(nil)
	if err != nil {
		return err
	}
	return da.db.Put(txIndexKey(seqid), data)
}

// // LoadState load get state from database
// func (da *Accessor) LoadState(addr types.Address) (*State, error) {
// 	data, dbErr := da.db.Get(stateKey(addr))
// 	if dbErr != nil {
// 		return nil, fmt.Errorf("can't get state from db, err: %v", dbErr)
// 	}
// 	var state State
// 	_, err := state.UnmarshalMsg(data)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &state, nil
// }

// // SaveState store the state data into db. Overwrite the data if it already exists.
// func (da *Accessor) SaveState(addr types.Address, state *State) error {
// 	data, err := state.MarshalMsg(nil)
// 	if err != nil {
// 		return err
// 	}
// 	return da.db.Put(stateKey(addr), data)
// }

// Save
