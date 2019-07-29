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
package p2p_message

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/bloom"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/types/msg"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"strings"
)

//go:generate msgp

const (
	BloomItemNumber = 3000
	HashFuncNum     = 8
)

type Message interface {
	msg.Message
	String() string //string is for logging
}

//msgp:tuple MessagePing
type MessagePing struct {
	Data []byte
}

//msgp:tuple MessagePong
type MessagePong struct {
	Data []byte
}

func (m *MessagePing) String() string {
	return fmt.Sprintf("ping")
}
func (m *MessagePong) String() string {
	return fmt.Sprintf("pong")
}

type HashTerminat [4]byte


type HashTerminats []HashTerminat

func (h HashTerminat)String ()string  {
	return hexutil.Encode(h[:])
}

func (h HashTerminats)String ()string  {
	var strs []string
	for _, v := range h {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

//msgp:tuple MessageSyncRequest
type MessageSyncRequest struct {
	Hashes    *common.Hashes
	HashTerminats  *HashTerminats
	Filter    *BloomFilter
	Height    *uint64
	RequestId uint32 //avoid msg drop
}

func (m *MessageSyncRequest) String() string {
	var str string
	if m.Filter != nil {
		str = fmt.Sprintf("count: %d", m.Filter.GetCount())
	}
	if m.Hashes!=nil {
		str += fmt.Sprintf("hash num %v", m.Hashes.String())
	}
	if m.HashTerminats!=nil {
		str += fmt.Sprintf("hashterminates %v ", m.HashTerminats.String())
	}
	str +=  fmt.Sprintf(" requestId %d  ", m.RequestId)
    return  str
}

//msgp:tuple MessageSyncResponse
type MessageSyncResponse struct {
	//RawTxs *RawTxs
	////SequencerIndex  []uint32
	//RawSequencers  *RawSequencers
	//RawCampaigns   *RawCampaigns
	//RawTermChanges *RawTermChanges
	RawTxs      *tx_types.TxisMarshaler
	RequestedId uint32 //avoid msg drop
}

func (m *MessageSyncResponse) Txis() types.Txis {
	return m.RawTxs.Txis()
}

func (m *MessageSyncResponse) Hashes() common.Hashes {
	var hashes common.Hashes
	if m.RawTxs != nil {
		for _, tx := range *m.RawTxs {
			if tx == nil {
				continue
			}
			hashes = append(hashes, tx.GetTxHash())
		}
	}

	return hashes
}

func (m *MessageSyncResponse) String() string {
	//for _,i := range m.SequencerIndex {
	//index = append(index ,fmt.Sprintf("%d",i))
	//}
	return fmt.Sprintf("txs: [%s],requestedId :%d", m.RawTxs.String(), m.RequestedId)
}

//msgp:tuple MessageNewTx
type MessageNewTx struct {
	RawTx *tx_types.RawTx
}

func (m *MessageNewTx) GetHash() *common.Hash {
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
	Data     []byte
	Count    uint32
	Capacity uint32
	filter   *bloom.BloomFilter
}

func (m *MessageNewTx) String() string {
	return m.RawTx.String()
}

//msgp:tuple MessageNewSequencer
type MessageNewSequencer struct {
	RawSequencer *tx_types.RawSequencer
	//Filter       *BloomFilter
	//Hop          uint8
}

func (m *MessageNewSequencer) GetHash() *common.Hash {
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
	return m.RawSequencer.String()
}

func (c *BloomFilter) GetCount() uint32 {
	if c == nil {
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
	if c.Capacity == 0 {
		c.filter = bloom.New(BloomItemNumber, HashFuncNum)
	} else {
		c.filter = bloom.New(uint(c.Capacity), HashFuncNum)
	}
	return c.filter.Decode(c.Data)
}

func NewDefaultBloomFilter() *BloomFilter {
	c := &BloomFilter{}
	c.filter = bloom.New(BloomItemNumber, HashFuncNum)
	c.Count = 0
	c.Capacity = 0 //0 if for default
	return c
}

func NewBloomFilter(m uint32) *BloomFilter {
	if m < BloomItemNumber {
		return NewDefaultBloomFilter()
	}
	c := &BloomFilter{}
	c.filter = bloom.New(uint(m), HashFuncNum)
	c.Count = 0
	c.Capacity = uint32(m)
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
	RawTxs *tx_types.RawTxs
}

func (m *MessageNewTxs) Txis() types.Txis {
	if m == nil {
		return nil
	}
	return m.RawTxs.Txis()
}

func (m *MessageNewTxs) Hashes() common.Hashes {
	var hashes common.Hashes
	if m.RawTxs == nil || len(*m.RawTxs) == 0 {
		return nil
	}
	for _, tx := range *m.RawTxs {
		if tx == nil {
			continue
		}
		hashes = append(hashes, tx.GetTxHash())
	}
	return hashes
}

func (m *MessageNewTxs) String() string {
	return m.RawTxs.String()
}

//msgp:tuple MessageTxsRequest
type MessageTxsRequest struct {
	Hashes    *common.Hashes
	SeqHash   *common.Hash
	Id        *uint64
	RequestId uint32 //avoid msg drop
}

func (m *MessageTxsRequest) String() string {
	return fmt.Sprintf("hashes: [%s], seqHash: %s, id : %d, requstId : %d", m.Hashes.String(), m.SeqHash.String(), m.Id, m.RequestId)
}

//msgp:tuple MessageTxsResponse
type MessageTxsResponse struct {
	//RawTxs         *RawTxs
	RawSequencer *tx_types.RawSequencer
	//RawCampaigns   *RawCampaigns
	//RawTermChanges *RawTermChanges
	RawTxs      *tx_types.TxisMarshaler
	RequestedId uint32 //avoid msg drop
}

func (m *MessageTxsResponse) String() string {
	return fmt.Sprintf("txs: [%s], Sequencer: %s, requestedId %d", m.RawTxs.String(), m.RawSequencer.String(), m.RequestedId)
}

func (m *MessageTxsResponse) Hashes() common.Hashes {
	var hashes common.Hashes
	if m.RawTxs == nil || len(*m.RawTxs) == 0 {
		return nil
	}
	for _, tx := range *m.RawTxs {
		if tx == nil {
			continue
		}
		hashes = append(hashes, tx.GetTxHash())
	}
	if m.RawSequencer != nil {
		hashes = append(hashes, m.RawSequencer.GetTxHash())
	}
	return hashes
}

//msgp:tuple MessageBodyData
type MessageBodyData struct {
	//RawTxs         *RawTxs
	//RawTermChanges *RawTermChanges
	//RawCampaigns   *RawCampaigns
	RawSequencer *tx_types.RawSequencer
	RawTxs       *tx_types.TxisMarshaler
}

func (m *MessageBodyData) ToTxis() types.Txis {
	var txis types.Txis
	if m.RawTxs != nil {
		txs := m.RawTxs.Txis()
		txis = append(txis, txs...)
	}
	if len(txis) == 0 {
		return nil
	}
	return txis
}

func (m *MessageBodyData) String() string {
	return fmt.Sprintf("txs: [%s], Sequencer: %s", m.RawTxs.String(), m.RawSequencer.String())
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
	Hash   *common.Hash // Block hash from which to retrieve headers (excludes Number)
	Number *uint64      // Block hash from which to retrieve headers (excludes Hash)
}

func (m *HashOrNumber) String() string {
	if m.Hash == nil {
		return fmt.Sprintf("hash: nil, number : %d ", *m.Number)
	}
	return fmt.Sprintf("hash: %s, number : %d", m.Hash.String(), m.Number)
}

//msgp:tuple MessageSequencerHeader
type MessageSequencerHeader struct {
	Hash   *common.Hash
	Number *uint64
}

func (m *MessageSequencerHeader) String() string {
	return fmt.Sprintf("hash: %s, number : %d", m.Hash.String(), m.Number)
}

//msgp:tuple MessageHeaderResponse
type MessageHeaderResponse struct {
	Headers     *tx_types.SequencerHeaders
	RequestedId uint32 //avoid msg drop
}

func (m *MessageHeaderResponse) String() string {
	return fmt.Sprintf("headers: [%s] reuqestedId :%d", m.Headers.String(), m.RequestedId)
}

//msgp:tuple MessageBodiesRequest
type MessageBodiesRequest struct {
	SeqHashes common.Hashes
	RequestId uint32 //avoid msg drop
}

func (m *MessageBodiesRequest) String() string {
	return m.SeqHashes.String() + fmt.Sprintf(" requestId :%d", m.RequestId)
}

//msgp:tuple MessageBodiesResponse
type MessageBodiesResponse struct {
	Bodies      []RawData
	RequestedId uint32 //avoid msg drop
}

func (m *MessageBodiesResponse) String() string {
	return fmt.Sprintf("bodies len : %d, reuqestedId :%d", len(m.Bodies), m.RequestedId)
}

//msgp:tuple RawData
type RawData []byte

//msgp:tuple MessageControl
type MessageControl struct {
	Hash *common.Hash
}

//msgp:tuple MessageGetMsg
type MessageGetMsg struct {
	Hash *common.Hash
}

type MessageDuplicate bool

func (m *MessageControl) String() string {
	if m == nil || m.Hash == nil {
		return ""
	}
	return m.Hash.String()
}

func (m *MessageGetMsg) String() string {
	if m == nil || m.Hash == nil {
		return ""
	}
	return m.Hash.String()
}

func (m *MessageDuplicate) String() string {
	return "duplicate"
}

//msgp:tuple MessageCampaign
type MessageCampaign struct {
	RawCampaign *tx_types.RawCampaign
}

func (m *MessageCampaign) String() string {
	return m.RawCampaign.String()
}

//msgp:tuple MessageTermChange
type MessageTermChange struct {
	RawTermChange *tx_types.RawTermChange
}

func (m *MessageTermChange) String() string {
	return m.RawTermChange.String()
}

//msgp:tuple MessageTermChangeResponse
type MessageTermChangeResponse struct {
	TermChange *tx_types.TermChange
	Id         uint32
}

//msgp:tuple MessageTermChangeRequest
type MessageTermChangeRequest struct {
	Id uint32
}

func (m MessageTermChangeResponse) String() string {
	return fmt.Sprintf("requst id %d , %v ", m.Id, m.TermChange)
}

func (m MessageTermChangeRequest) String() string {
	return fmt.Sprintf("requst id %d ", m.Id)
}

//msgp:tuple MessageConsensusDkgSigSets
type MessageConsensusDkgSigSets struct {
	PkBls     []byte
	PublicKey []byte
	Signature []byte
	TermId    uint64
}

//msgp:tuple MessageConsensusDkgGenesisPublicKey
type MessageConsensusDkgGenesisPublicKey struct {
	DkgPublicKey []byte
	PublicKey    []byte
	Signature    []byte
	TermId       uint64
}

func (m *MessageConsensusDkgGenesisPublicKey) SignatureTargets() []byte {
	return m.DkgPublicKey
}

func (m *MessageConsensusDkgGenesisPublicKey) String() string {
	return fmt.Sprintf("DkgGenesisPublicKey  len %d ", len(m.DkgPublicKey))
}

//msgp:tuple MessageConsensusDkgDeal
type MessageConsensusDkgDeal struct {
	Id        uint32
	Data      []byte
	PublicKey []byte
	Signature []byte
	TermId    uint64
}

func (m *MessageConsensusDkgSigSets) SignatureTargets() []byte {
	w := types.NewBinaryWriter()
	w.Write(m.PkBls)
	return w.Bytes()
}

func (m *MessageConsensusDkgSigSets) String() string {
	return "dkgSigsets" + fmt.Sprintf("len %d", len(m.PkBls))
}

func (m *MessageConsensusDkgDeal) SignatureTargets() []byte {
	w := types.NewBinaryWriter()
	d := []byte(m.Data)
	w.Write(d, m.Id)
	return w.Bytes()
}

func (m MessageConsensusDkgDeal) String() string {
	var pkstr string
	if len(m.PublicKey) > 10 {
		pkstr = hexutil.Encode(m.PublicKey[:5])
	}
	return "dkg " + fmt.Sprintf(" id %d , len %d  tid %d", m.Id, len(m.Data), m.TermId) + " pk-" + pkstr
}

//msgp:tuple MessageConsensusDkgDealResponse
type MessageConsensusDkgDealResponse struct {
	Id        uint32
	Data      []byte
	PublicKey []byte
	Signature []byte
	TermId    uint64
}

func (m MessageConsensusDkgDealResponse) String() string {
	var pkstr string
	if len(m.PublicKey) > 10 {
		pkstr = hexutil.Encode(m.PublicKey[:5])
	}
	return "dkgresponse " + fmt.Sprintf(" id %d , len %d  tid %d", m.Id, len(m.Data), m.TermId) + " pk-" + pkstr
}

func (m *MessageConsensusDkgDealResponse) SignatureTargets() []byte {
	d := []byte(m.Data)
	w := types.NewBinaryWriter()
	w.Write(d, m.Id)
	return w.Bytes()
}

//msgp:tuple MessageNewArchive
type MessageNewArchive struct {
	Archive *tx_types.Archive
}

func (m *MessageNewArchive) String() string {
	if m.Archive == nil {
		return "nil"
	}
	return m.Archive.String()
}

//msgp:tuple MessageNewActionTx
type MessageNewActionTx struct {
	ActionTx *tx_types.ActionTx
}

func (m *MessageNewActionTx) String() string {
	if m.ActionTx == nil {
		return "nil"
	}
	return m.ActionTx.String()
}
