package types

import (
	"fmt"
	"github.com/annchain/OG/common/math"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/tinylib/msgp/msgp"
	"strings"
)

var Signer ISigner

type ISigner interface {
	AddressFromPubKeyBytes(pubKey []byte) Address
}

//go:generate msgp

type Message interface {
	msgp.MarshalSizer
	msgp.Unmarshaler
	msgp.Decodable
	msgp.Encodable
	String() string //string is for logging ,
}

type MessageWithFilter interface {
	Message
	AddItem(item []byte) error
	LookUpItem(item []byte) (bool, error)
	MarshalMsgWithOutFilter([]byte) ([]byte, error)
	SetFilter(*CuckooFilter)
	GetFilter() *CuckooFilter
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
	RequestId uint32 //avoid msg drop
}

func (m *MessageSyncRequest) String() string {
	return HashesToString(m.Hashes) + fmt.Sprintf(" requestId %d", m.RequestId)
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
	RawTx        *RawTx
	CuckooFilter *CuckooFilter
}

//msgp:tuple CuckooFilter
type CuckooFilter struct {
	Data   []byte
	filter *cuckoofilter.CuckooFilter
}

func (m *MessageNewTx) String() string {
	return m.RawTx.String()
}

//msgp:tuple MessageNewSequencer
type MessageNewSequencer struct {
	RawSequencer *RawSequencer
	CuckooFilter *CuckooFilter
}

func (m *MessageNewSequencer) String() string {
	return m.RawSequencer.String()
}

func (m *MessageNewSequencer) AddItem(item []byte) error {
	return m.CuckooFilter.AddItem(item)
}

func (m *MessageNewSequencer) LookUpItem(item []byte) (bool, error) {
	return m.CuckooFilter.LookUpItem(item)
}

func (m *MessageNewSequencer) DelFilter() {
	m.CuckooFilter = nil
}

func (m *MessageNewSequencer) MarshalMsgWithOutFilter(b []byte) ([]byte, error) {
	newMsg := &MessageNewSequencer{
		RawSequencer: m.RawSequencer,
	}
	return newMsg.MarshalMsg(b)
}

func (m *MessageNewSequencer) SetFilter(c *CuckooFilter) {
	m.CuckooFilter = c
}

func (m *MessageNewSequencer) GetFilter() *CuckooFilter {
	return m.CuckooFilter
}

func (m *MessageNewTx) AddItem(item []byte) error {
	return m.CuckooFilter.AddItem(item)
}

func (m *MessageNewTx) LookUpItem(item []byte) (bool, error) {
	return m.CuckooFilter.LookUpItem(item)
}

func (m *MessageNewTx) DelFilter() {
	m.CuckooFilter = nil
}

func (m *MessageNewTx) MarshalMsgWithOutFilter(b []byte) ([]byte, error) {
	newMsg := &MessageNewTx{
		RawTx: m.RawTx,
	}
	return newMsg.MarshalMsg(b)
}

func (m *MessageNewTx) SetFilter(c *CuckooFilter) {
	m.CuckooFilter = c
}

func (m *MessageNewTx) GetFilter() *CuckooFilter {
	return m.CuckooFilter
}

func (c *CuckooFilter) AddItem(item []byte) error {
	if c == nil {
		c = &CuckooFilter{}
	}
	if c.filter == nil {
		if len(c.Data) != 0 {
			cf, err := cuckoofilter.Decode(c.Data)
			if err != nil {
				return err
			}
			c.filter = cf
		}
	}
	c.filter.Insert(item)
	c.Data = c.filter.Encode()
	return nil
}

func (c *CuckooFilter) LookUpItem(item []byte) (bool, error) {
	if c == nil || c.filter == nil {
		return false, nil
	}
	return c.filter.Lookup(item), nil
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
