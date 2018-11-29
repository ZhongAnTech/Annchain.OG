package types

import (
	"fmt"
	"github.com/Metabdulla/bloom"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/common/msgp"
	"strings"
)
//go:generate msgp

var Signer ISigner

const (
	BloomItemNumber = 3000
	HashFuncNum     = 8
)

type ISigner interface {
	AddressFromPubKeyBytes(pubKey []byte) Address
}



type Message interface {
  	 msgp.Message
	String() string //string is for logging
}

type MessagePing struct {
	Data []byte
}

type MessagePong struct {
	Data []byte
}

func (m *MessagePing) String() string {
	return fmt.Sprintf("ping")
}
func (m *MessagePong) String() string {
	return fmt.Sprintf("pong")
}

//msgp:tuple MessageSyncRequest
type MessageSyncRequest struct {
	Hashes    []Hash
	Filter    *BloomFilter
	RequestId uint32 //avoid msg drop
}

func (m *MessageSyncRequest) String() string {
	return fmt.Sprintf(" requestId %d", m.RequestId) + fmt.Sprintf("count: %d", m.Filter.GetCount())
}

//msgp:tuple MessageSyncResponse
type MessageSyncResponse struct {
	RawTxs        []*RawTx
	RawSequencers []*RawSequencer
	RequestedId   uint32 //avoid msg drop
}

func (m *MessageSyncResponse) String() string {
	return fmt.Sprintf("txs: [%s], seqs: [%s] ,requestedId :%d", RawTxsToString(m.RawTxs), RawSeqsToString(m.RawSequencers), m.RequestedId)
}

//msgp:tuple MessageNewTx
type MessageNewTx struct {
	RawTx *RawTx
}

func (m *MessageNewTx) GetHash() *Hash {
	if m == nil {
		return nil
	}
	if m.RawTx == nil {
		return nil
	}
	hash := m.RawTx.GetTxHash()
	return &hash

}

//msgp:tuple BloomFilter
type BloomFilter struct {
	Data   []byte
	Count   uint32
	filter *bloom.BloomFilter
}

func (m *MessageNewTx) String() string {
	return m.RawTx.String()
}

//msgp:tuple MessageNewSequencer
type MessageNewSequencer struct {
	RawSequencer *RawSequencer
	Filter    *BloomFilter
	Hop       uint8
}

func (m *MessageNewSequencer) GetHash() *Hash {
	if m == nil {
		return nil
	}
	if m.RawSequencer == nil {
		return nil
	}
	hash := m.RawSequencer.GetTxHash()
	return &hash

}

func (m *MessageNewSequencer) String() string {
	return m.RawSequencer.String()+fmt.Sprintf("  hop :%d , filterCount :%d",m.Hop,m.Filter.GetCount())
}

func (c*BloomFilter)GetCount()uint32 {
	if c ==nil {
		return 0
	}
	return c.Count
}

func (c *BloomFilter) Encode() error {
	var err error
	c.Data, err = c.filter.Encode()
	return err
}

func (c *BloomFilter) Decode() error {
	c.filter = bloom.New(BloomItemNumber, HashFuncNum)
	return c.filter.Decode(c.Data)
}

func NewDefaultBloomFilter() *BloomFilter {
	c := &BloomFilter{}
	c.filter = bloom.New(BloomItemNumber, HashFuncNum)
	c.Count = 0
	return c
}

func (c *BloomFilter) AddItem(item []byte) {
	c.filter.Add(item)
	c.Count++
}


func (c *BloomFilter) LookUpItem(item []byte) (bool, error) {
	if c == nil || c.filter == nil {
		return false, nil
	}
	return c.filter.Test(item), nil
}


//msgp:tuple MessageNewTxs
type MessageNewTxs struct {
	RawTxs []*RawTx
}

func (m *MessageNewTxs) String() string {
	return RawTxsToString(m.RawTxs)
}

//msgp:tuple MessageTxsRequest
type MessageTxsRequest struct {
	Hashes    []Hash
	SeqHash   *Hash
	Id        uint64
	RequestId uint32 //avoid msg drop
}

func (m *MessageTxsRequest) String() string {
	return fmt.Sprintf("hashes: [%s], seqHash: %s, id : %d, requstId : %d", HashesToString(m.Hashes), m.SeqHash.String(), m.Id, m.RequestId)
}

//msgp:tuple MessageTxsResponse
type MessageTxsResponse struct {
	RawTxs       []*RawTx
	RawSequencer *RawSequencer
	RequestedId  uint32 //avoid msg drop
}

func (m *MessageTxsResponse) String() string {
	return fmt.Sprintf("txs: [%s], Sequencer: %s, requestedId %d", RawTxsToString(m.RawTxs), m.RawSequencer.String(), m.RequestedId)
}

// getBlockHeadersData represents a block header query.
//msgp:tuple MessageHeaderRequest
type MessageHeaderRequest struct {
	Origin    HashOrNumber // Block from which to retrieve headers
	Amount    uint64       // Maximum number of headers to retrieve
	Skip      uint64       // Blocks to skip between consecutive headers
	Reverse   bool         // Query direction (false = rising towards latest, true = falling towards genesis)
	RequestId uint32       //avoid msg drop
}

func (m *MessageHeaderRequest) String() string {
	return fmt.Sprintf("Origin: [%s],amount : %d ,skip : %d, reverse : %v, requestId :%d", m.Origin.String(), m.Amount, m.Skip, m.Reverse, m.RequestId)
}

// hashOrNumber is a combined field for specifying an origin block.
//msgp:tuple HashOrNumber
type HashOrNumber struct {
	Hash   Hash   // Block hash from which to retrieve headers (excludes Number)
	Number uint64 // Block hash from which to retrieve headers (excludes Hash)
}

func (m *HashOrNumber) String() string {
	return fmt.Sprintf("hash: %s, number : %d", m.Hash.String(), m.Number)
}

//msgp:tuple MessageSequencerHeader
type MessageSequencerHeader struct {
	Hash   *Hash
	Number uint64
}

func (m *MessageSequencerHeader) String() string {
	return fmt.Sprintf("hash: %s, number : %d", m.Hash.String(), m.Number)
}

//msgp:tuple MessageHeaderResponse
type MessageHeaderResponse struct {
	RawSequencers []*RawSequencer
	RequestedId   uint32 //avoid msg drop
}

func (m *MessageHeaderResponse) String() string {
	return fmt.Sprintf("seqs: [%s] reuqestedId :%d", RawSeqsToString(m.RawSequencers), m.RequestedId)
}

//msgp:tuple MessageBodiesRequest
type MessageBodiesRequest struct {
	SeqHashes []Hash
	RequestId uint32 //avoid msg drop
}

func (m *MessageBodiesRequest) String() string {
	return HashesToString(m.SeqHashes) + fmt.Sprintf(" requestId :%d", m.RequestId)
}

//msgp:tuple MessageBodiesResponse
type MessageBodiesResponse struct {
	Bodies      []RawData
	RequestedId uint32 //avoid msg drop
}

func (m *MessageBodiesResponse) String() string {
	return fmt.Sprintf("bodies len : %d, reuqestedId :%d", len(m.Bodies), m.RequestedId)
}

type RawData []byte

// compress data ,for p2p  , small size
type RawTx struct {
	TxBase
	To    Address
	Value *math.BigInt
}

type RawSequencer struct {
	TxBase
	Id                uint64 `msgp:"id"`
	ContractHashOrder []Hash `msgp:"contractHashOrder"`
}

func (t *RawTx) Tx() *Tx {
	if t == nil {
		return nil
	}
	tx := &Tx{
		TxBase: t.TxBase,
		To:     t.To,
		Value:  t.Value,
	}
	tx.From = Signer.AddressFromPubKeyBytes(tx.PublicKey)
	return tx
}

func (t *RawSequencer) Sequencer() *Sequencer {
	if t == nil {
		return nil
	}
	tx := &Sequencer{
		TxBase:            t.TxBase,
		Id:                t.Id,
		ContractHashOrder: t.ContractHashOrder,
	}
	tx.Issuer = Signer.AddressFromPubKeyBytes(tx.PublicKey)
	return tx
}

func (t *RawTx) String() string {
	return fmt.Sprintf("%s-%d-RawTx", t.TxBase.String(), t.AccountNonce)
}

func (t *RawSequencer) String() string {
	return fmt.Sprintf("%s-%d-id_%d-Seq", t.TxBase.String(), t.AccountNonce, t.Id)
}

func RawTxsToTxs(rawTxs []*RawTx) []*Tx {
	if len(rawTxs) == 0 {
		return nil
	}
	var txs []*Tx
	for _, v := range rawTxs {
		tx := v.Tx()
		txs = append(txs, tx)
	}
	return txs
}
func RawSequencerToSeqSequencers(rawSeqs []*RawSequencer) []*Sequencer {
	if len(rawSeqs) == 0 {
		return nil
	}
	var seqs []*Sequencer
	for _, v := range rawSeqs {
		seq := v.Sequencer()
		seqs = append(seqs, seq)
	}
	return seqs
}

func TxsToRawTxs(txs []*Tx) []*RawTx {
	if len(txs) == 0 {
		return nil
	}
	var rawTxs []*RawTx
	for _, v := range txs {
		rasTx := v.RawTx()
		rawTxs = append(rawTxs, rasTx)
	}
	return rawTxs
}

func SequencersToRawSequencers(seqs []*Sequencer) []*RawSequencer {
	if len(seqs) == 0 {
		return nil
	}
	var rawSeqs []*RawSequencer
	for _, v := range seqs {
		rasSeq := v.RawSequencer()
		rawSeqs = append(rawSeqs, rasSeq)
	}
	return rawSeqs
}

func RawTxsToString(txs []*RawTx) string {
	var strs []string
	for _, v := range txs {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func RawSeqsToString(seqs []*RawSequencer) string {
	var strs []string
	for _, v := range seqs {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}
