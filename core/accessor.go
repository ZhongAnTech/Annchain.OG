// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/consensus/campaign"
	"github.com/annchain/OG/og/archive"
	"github.com/annchain/OG/og/protocol/ogmessage"
	archive2 "github.com/annchain/OG/og/protocol/ogmessage/archive"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"strconv"
	"sync"
)

var (
	prefixGenesisKey   = []byte("genesis")
	prefixLatestSeqKey = []byte("latestseq")

	prefixReceiptKey = []byte("rp")

	prefixTransactionKey = []byte("tx")
	prefixTxHashFlowKey  = []byte("fl")

	contentPrefixTransaction = []byte("cptx")
	contentPrefixSequencer   = []byte("cpsq")
	contentPrefixCampaign    = []byte("cpcp")
	contentPrefixTermChg     = []byte("cptc")
	contentPrefixArchive     = []byte("cpac")
	contentPrefixActionTx    = []byte("cpax")

	prefixAddrLatestNonceKey = []byte("aln")

	prefixSeqHeightKey = []byte("sh")
	prefixTxIndexKey   = []byte("ti")

	prefixAddressBalanceKey = []byte("ba")

	prefixConfirmtime = []byte("cf")

	prefixStateRootKey = []byte("stateroot")
)

// TODO encode uint to specific length bytes

func genesisKey() []byte {
	return prefixGenesisKey
}

func latestSequencerKey() []byte {
	return prefixLatestSeqKey
}

func receiptKey(seqID uint64) []byte {
	return append(prefixReceiptKey, encodeUint64(seqID)...)
}

func transactionKey(hash common.Hash) []byte {
	return append(prefixTransactionKey, hash.ToBytes()...)
}

func confirmTimeKey(SeqHeight uint64) []byte {
	return append(prefixConfirmtime, encodeUint64(SeqHeight)...)
}

func txHashFlowKey(addr common.Address, nonce uint64) []byte {
	keybody := append(addr.ToBytes(), encodeUint64(nonce)...)
	return append(prefixTxHashFlowKey, keybody...)
}

func addrLatestNonceKey(addr common.Address) []byte {
	return append(prefixAddrLatestNonceKey, addr.ToBytes()...)
}

func addressBalanceKey(addr common.Address) []byte {
	return append(prefixAddressBalanceKey, addr.ToBytes()...)
}

func seqHeightKey(seqID uint64) []byte {
	return append(prefixSeqHeightKey, encodeUint64(seqID)...)
}

func txIndexKey(seqID uint64) []byte {
	return append(prefixTxIndexKey, encodeUint64(seqID)...)
}

func stateRootKey() []byte {
	return prefixStateRootKey
}

type Accessor struct {
	db ogdb.Database
}

func NewAccessor(db ogdb.Database) *Accessor {
	return &Accessor{
		db: db,
	}
}

type Putter struct {
	ogdb.Batch
	wg                   *sync.WaitGroup
	writeConcurrenceChan chan bool
	err                  error
	mu                   sync.RWMutex
}

func (ac *Accessor) NewBatch() *Putter {
	return &Putter{Batch: ac.db.NewBatch(),
		writeConcurrenceChan: make(chan bool, 100), wg: &sync.WaitGroup{}}
}

func (da *Accessor) put(putter *Putter, key []byte, data []byte) error {
	if putter == nil || putter.Batch == nil {
		err := da.db.Put(key, data)
		if err != nil {
			log.Errorf("write data to db batch err: %v", err)
		}
		return err
	}
	return putter.Put(key, data)
}

func (p Putter) Write() error {
	if p.Batch == nil {
		return nil
	}
	if p.err != nil {
		return p.err
	}
	p.wg.Wait()
	//fmt.Println(p.wg,&p.wg,"end", p, p.Batch.ValueSize())
	defer p.Batch.Reset()
	return p.Batch.Write()
}

func (p *Putter) put(key []byte, data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	//fmt.Println(p.wg,&p.wg,"start haha", p, p.Batch.ValueSize())
	err := p.Batch.Put(key, data)
	if err != nil {
		log.Errorf("write tx to db batch err: %v", err)
	}
	if err != nil {
		p.err = err
	}
	<-p.writeConcurrenceChan
	//fmt.Println("hihi", time.Now())
	//fmt.Println(p.wg, "inside",&p.wg,p,p.Batch.ValueSize())
	p.wg.Done()
	//fmt.Println("done", time.Now())
	//fmt.Println(p.wg, "inside",&p.wg,p,p.Batch.ValueSize())
}

func (p *Putter) Put(key []byte, data []byte) error {
	if p.err != nil {
		return p.err
	}
	put := func() {
		p.put(key, data)
	}
	p.writeConcurrenceChan <- true
	p.wg.Add(1)
	goroutine.New(put)
	return p.err
}

// ReadGenesis get genesis sequencer from db.
// return nil if there is no genesis.
func (da *Accessor) ReadGenesis() *ogmessage.Sequencer {
	data, _ := da.db.Get(genesisKey())
	if len(data) == 0 {
		return nil
	}
	var seq ogmessage.Sequencer
	_, err := seq.UnmarshalMsg(data)
	if err != nil {
		return nil
	}
	return &seq
}

// WriteGenesis writes geneis into db.
func (da *Accessor) WriteGenesis(genesis *ogmessage.Sequencer) error {
	data, err := genesis.MarshalMsg(nil)
	if err != nil {
		return err
	}
	return da.put(nil, genesisKey(), data)
}

// ReadLatestSequencer get latest sequencer from db.
// return nil if there is no sequencer.
func (da *Accessor) ReadLatestSequencer() *ogmessage.Sequencer {
	data, _ := da.db.Get(latestSequencerKey())
	if len(data) == 0 {
		return nil
	}
	var seq ogmessage.Sequencer
	_, err := seq.UnmarshalMsg(data)
	if err != nil {
		return nil
	}
	return &seq
}

// WriteGenesis writes latest sequencer into db.
func (da *Accessor) WriteLatestSequencer(putter *Putter, seq *ogmessage.Sequencer) error {
	data, err := seq.MarshalMsg(nil)
	if err != nil {
		return err
	}
	return da.put(putter, latestSequencerKey(), data)
}

// ReadLastStateRoot read latest state root from db.
// TODO
// this is a temp function. The latest state root should be stored
// in latest sequencer.
func (da *Accessor) ReadLastStateRoot() common.Hash {
	rootbyte, _ := da.db.Get(stateRootKey())
	return common.BytesToHash(rootbyte)
}

// WriteLastStateRoot write latest state root into db.
// TODO
// this is a temp function. The latest state root should be stored
// in latest sequencer.
func (da *Accessor) WriteLastStateRoot(putter *Putter, root common.Hash) error {
	return da.db.Put(stateRootKey(), root.ToBytes())
}

// ReadTransaction get tx or sequencer from ogdb.
func (da *Accessor) ReadTransaction(hash common.Hash) ogmessage.Txi {
	data, _ := da.db.Get(transactionKey(hash))
	if len(data) == 0 {
		return nil
	}
	prefixLen := len(contentPrefixTransaction)
	prefix := data[:prefixLen]
	data = data[prefixLen:]
	if bytes.Equal(prefix, contentPrefixTransaction) {
		var tx archive2.Tx
		_, err := tx.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal tx error")
			return nil
		}
		return &tx
	}
	if bytes.Equal(prefix, contentPrefixSequencer) {
		var sq ogmessage.Sequencer
		_, err := sq.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal seq error")
			return nil
		}
		return &sq
	}
	if bytes.Equal(prefix, contentPrefixCampaign) {
		var cp campaign.Campaign
		_, err := cp.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal camp error")
			return nil
		}
		return &cp
	}
	if bytes.Equal(prefix, contentPrefixTermChg) {
		var tc campaign.TermChange
		_, err := tc.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal termchg error")
			return nil
		}
		return &tc
	}
	if bytes.Equal(prefix, contentPrefixArchive) {
		var ac archive.Archive
		_, err := ac.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal archive error")
			return nil
		}
		return &ac
	}
	if bytes.Equal(prefix, contentPrefixActionTx) {
		var ac archive2.ActionTx
		_, err := ac.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal archive error")
			return nil
		}
		return &ac
	}

	return nil
}

// ReadTxByNonce get tx from db by sender's address and nonce.
func (da *Accessor) ReadTxByNonce(addr common.Address, nonce uint64) ogmessage.Txi {
	data, _ := da.db.Get(txHashFlowKey(addr, nonce))
	if len(data) == 0 {
		return nil
	}
	hash := common.BytesToHash(data)
	return da.ReadTransaction(hash)
}

// WriteTxHashByNonce writes tx hash into db and construct key with address and nonce.
func (da *Accessor) WriteTxHashByNonce(putter *Putter, addr common.Address, nonce uint64, hash common.Hash) error {
	data := hash.ToBytes()
	var err error
	key := txHashFlowKey(addr, nonce)
	err = da.put(putter, key, data)
	if err != nil {
		return fmt.Errorf("write tx hash flow to db err: %v", err)
	}
	return nil
}

// ReadAddrLatestNonce get latest nonce of an address
func (da *Accessor) ReadAddrLatestNonce(addr common.Address) (uint64, error) {
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
func (da *Accessor) HasAddrLatestNonce(addr common.Address) (bool, error) {
	return da.db.Has(addrLatestNonceKey(addr))
}
func (da *Accessor) writeConfirmTime(cf *types.ConfirmTime) error {
	data, err := cf.MarshalMsg(nil)
	if err != nil {
		return err
	}
	err = da.db.Put(confirmTimeKey(cf.SeqHeight), data)
	if err != nil {
		return fmt.Errorf("write tx to db batch err: %v", err)
	}

	return nil
}

func (da *Accessor) readConfirmTime(SeqHeight uint64) *types.ConfirmTime {
	data, _ := da.db.Get(confirmTimeKey(SeqHeight))
	if len(data) == 0 {
		return nil
	}
	var cf types.ConfirmTime
	_, err := cf.UnmarshalMsg(data)
	if err != nil || cf.SeqHeight != SeqHeight {
		return nil
	}
	return &cf
}

// WriteReceipts write a receipt map into db.
func (da *Accessor) WriteReceipts(putter *Putter, seqID uint64, receipts ReceiptSet) error {
	data, err := receipts.MarshalMsg(nil)
	if err != nil {
		return fmt.Errorf("marshal seq%d's receipts err: %v", seqID, err)
	}
	err = da.put(putter, receiptKey(seqID), data)
	if err != nil {
		return fmt.Errorf("write seq%d's receipts err: %v", seqID, err)
	}
	return nil
}

// ReadReceipt try get receipt by tx hash and seqID.
func (da *Accessor) ReadReceipt(seqID uint64, hash common.Hash) *Receipt {
	bundleBytes, _ := da.db.Get(receiptKey(seqID))
	if len(bundleBytes) == 0 {
		return nil
	}
	var receipts ReceiptSet
	_, err := receipts.UnmarshalMsg(bundleBytes)
	if err != nil {
		return nil
	}
	receipt, ok := receipts[hash.Hex()]
	if !ok {
		return nil
	}
	return receipt
}

// WriteTransaction write the tx or sequencer into ogdb.
func (da *Accessor) WriteTransaction(putter *Putter, tx ogmessage.Txi) error {
	var prefix, data []byte
	var err error

	// write tx
	switch tx := tx.(type) {
	case *archive2.Tx:
		prefix = contentPrefixTransaction
		data, err = tx.MarshalMsg(nil)
	case *ogmessage.Sequencer:
		prefix = contentPrefixSequencer
		data, err = tx.MarshalMsg(nil)
	case *campaign.Campaign:
		prefix = contentPrefixCampaign
		data, err = tx.MarshalMsg(nil)
	case *campaign.TermChange:
		prefix = contentPrefixTermChg
		data, err = tx.MarshalMsg(nil)
	case *archive.Archive:
		prefix = contentPrefixArchive
		data, err = tx.MarshalMsg(nil)
	case *archive2.ActionTx:
		prefix = contentPrefixActionTx
		data, err = tx.MarshalMsg(nil)
	default:
		return fmt.Errorf("unknown tx type, must be *Tx, *Sequencer, *Campaign, *TermChange")
	}
	if err != nil {
		return fmt.Errorf("marshal tx %s err: %v", tx.GetTxHash(), err)
	}
	data = append(prefix, data...)
	key := transactionKey(tx.GetTxHash())
	da.put(putter, key, data)

	return nil
}

// DeleteTransaction delete the tx or sequencer.
func (da *Accessor) DeleteTransaction(hash common.Hash) error {
	return da.db.Delete(transactionKey(hash))
}

// ReadBalance get the balance of an address.
func (da *Accessor) ReadBalance(addr common.Address) *math.BigInt {
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
func (da *Accessor) SetBalance(putter *Putter, addr common.Address, value *math.BigInt) error {
	if value.Value.Abs(value.Value).Cmp(value.Value) != 0 {
		return fmt.Errorf("the value of the balance must be positive!")
	}
	data, err := value.MarshalMsg(nil)
	if err != nil {
		return err
	}
	key := addressBalanceKey(addr)
	return da.put(putter, key, data)
}

// DeleteBalance delete the balance of an address.
func (da *Accessor) DeleteBalance(addr common.Address) error {
	return da.db.Delete(addressBalanceKey(addr))
}

// AddBalance adds an amount of value to the address balance. Note that AddBalance
// doesn't hold any locks so upper level program must manage this.
func (da *Accessor) AddBalance(putter *Putter, addr common.Address, amount *math.BigInt) error {
	if amount.Value.Abs(amount.Value).Cmp(amount.Value) != 0 {
		return fmt.Errorf("add amount must be positive!")
	}
	balance := da.ReadBalance(addr)
	// no balance exists
	if balance == nil {
		return da.SetBalance(putter, addr, amount)
	}
	newBalanceValue := balance.Value.Add(balance.Value, amount.Value)
	return da.SetBalance(putter, addr, &math.BigInt{Value: newBalanceValue})
}

// SubBalance subs an amount of value to the address balance. Note that SubBalance
// doesn't hold any locks so upper level program must manage this.
func (da *Accessor) SubBalance(putter *Putter, addr common.Address, amount *math.BigInt) error {
	if amount.Value.Abs(amount.Value).Cmp(amount.Value) != 0 {
		return fmt.Errorf("add amount must be positive!")
	}
	balance := da.ReadBalance(addr)
	// no balance exists
	if balance == nil {
		return fmt.Errorf("address %s has no balance yet, cannot process sub", addr)
	}
	if balance.Value.Cmp(amount.Value) == -1 {
		return fmt.Errorf("address %s has no enough balance to sub. balance: %d, sub amount: %d",
			addr, balance.GetInt64(), amount.GetInt64())
	}
	newBalanceValue := balance.Value.Sub(balance.Value, amount.Value)
	return da.SetBalance(putter, addr, &math.BigInt{Value: newBalanceValue})
}

// ReadSequencerByHeight get sequencer from db by sequencer id.
func (da *Accessor) ReadSequencerByHeight(SeqHeight uint64) (*ogmessage.Sequencer, error) {
	data, _ := da.db.Get(seqHeightKey(SeqHeight))
	if len(data) == 0 {
		return nil, fmt.Errorf("sequencer with SeqHeight %d not found", SeqHeight)
	}
	var seq ogmessage.Sequencer
	_, err := seq.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}
	return &seq, nil
}

// WriteSequencerByHeight stores the sequencer into db and indexed by its id.
func (da *Accessor) WriteSequencerByHeight(putter *Putter, seq *ogmessage.Sequencer) error {
	data, err := seq.MarshalMsg(nil)
	if err != nil {
		return err
	}
	key := seqHeightKey(seq.Height)
	return da.put(putter, key, data)
}

// ReadIndexedTxHashs get a list of txs that is confirmed by the sequencer that
// holds the id 'SeqHeight'.
func (da *Accessor) ReadIndexedTxHashs(SeqHeight uint64) (*common.Hashes, error) {
	data, _ := da.db.Get(txIndexKey(SeqHeight))
	if len(data) == 0 {
		return nil, fmt.Errorf("tx hashs with seq height %d not found", SeqHeight)
	}
	var hashs common.Hashes
	_, err := hashs.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}
	return &hashs, nil
}

// WriteIndexedTxHashs stores a list of tx hashs. These related hashs are all
// confirmed by sequencer that holds the id 'SeqHeight'.
func (da *Accessor) WriteIndexedTxHashs(putter *Putter, SeqHeight uint64, hashs *common.Hashes) error {
	data, err := hashs.MarshalMsg(nil)
	if err != nil {
		return err
	}
	key := txIndexKey(SeqHeight)
	return da.put(putter, key, data)
}

/**
Components
*/

func encodeUint64(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}

func encodeInt32(n int32) []byte {
	return common.ByteInt32(n)
}
