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
package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/annchain/OG/common/bloom"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/msg"
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

//msgp:tuple MessageSyncRequest
type MessageSyncRequest struct {
	Hashes    *Hashes
	Filter    *BloomFilter
	Height    *uint64
	RequestId uint32 //avoid msg drop
}

func (m *MessageSyncRequest) String() string {
	if m.Filter != nil {
		return fmt.Sprintf(" requestId %d  height: %v ", m.RequestId, m.Height) + fmt.Sprintf("count: %d", m.Filter.GetCount())
	}
	return m.Hashes.String() + fmt.Sprintf(" requestId %d  ", m.RequestId)

}

//msgp:tuple MessageSyncResponse
type MessageSyncResponse struct {
	//RawTxs *RawTxs
	////SequencerIndex  []uint32
	//RawSequencers  *RawSequencers
	//RawCampaigns   *RawCampaigns
	//RawTermChanges *RawTermChanges
	RawTxs      *TxisMarshaler
	RequestedId uint32 //avoid msg drop
}

func (m *MessageSyncResponse) Txis() Txis {
	return m.RawTxs.Txis()
}

func (m *MessageSyncResponse) Hashes() Hashes {
	var hashes Hashes
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
	Count  uint32
	filter *bloom.BloomFilter
}

func (m *MessageNewTx) String() string {
	return m.RawTx.String()
}

//msgp:tuple MessageNewSequencer
type MessageNewSequencer struct {
	RawSequencer *RawSequencer
	//Filter       *BloomFilter
	//Hop          uint8
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
	RawTxs *RawTxs
}

func (m *MessageNewTxs) Txis() Txis {
	return m.Txis()
}

func (m *MessageNewTxs) Hashes() Hashes {
	var hashes Hashes
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
	Hashes    *Hashes
	SeqHash   *Hash
	Id        *uint64
	RequestId uint32 //avoid msg drop
}

func (m *MessageTxsRequest) String() string {
	return fmt.Sprintf("hashes: [%s], seqHash: %s, id : %d, requstId : %d", m.Hashes.String(), m.SeqHash.String(), m.Id, m.RequestId)
}

//msgp:tuple MessageTxsResponse
type MessageTxsResponse struct {
	//RawTxs         *RawTxs
	RawSequencer *RawSequencer
	//RawCampaigns   *RawCampaigns
	//RawTermChanges *RawTermChanges
	RawTxs      *TxisMarshaler
	RequestedId uint32 //avoid msg drop
}

func (m *MessageTxsResponse) String() string {
	return fmt.Sprintf("txs: [%s], Sequencer: %s, requestedId %d", m.RawTxs.String(), m.RawSequencer.String(), m.RequestedId)
}

func (m *MessageTxsResponse) Hashes() Hashes {
	var hashes Hashes
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
	RawSequencer *RawSequencer
	RawTxs       *TxisMarshaler
}

func (m *MessageBodyData) ToTxis() Txis {
	var txis Txis
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
	Hash   *Hash   // Block hash from which to retrieve headers (excludes Number)
	Number *uint64 // Block hash from which to retrieve headers (excludes Hash)
}

func (m *HashOrNumber) String() string {
	if m.Hash == nil {
		return fmt.Sprintf("hash: nil, number : %d ", *m.Number)
	}
	return fmt.Sprintf("hash: %s, number : %d", m.Hash.String(), m.Number)
}

//msgp:tuple MessageSequencerHeader
type MessageSequencerHeader struct {
	Hash   *Hash
	Number *uint64
}

func (m *MessageSequencerHeader) String() string {
	return fmt.Sprintf("hash: %s, number : %d", m.Hash.String(), m.Number)
}

//msgp:tuple MessageHeaderResponse
type MessageHeaderResponse struct {
	Headers     *SequencerHeaders
	RequestedId uint32 //avoid msg drop
}

func (m *MessageHeaderResponse) String() string {
	return fmt.Sprintf("headers: [%s] reuqestedId :%d", m.Headers.String(), m.RequestedId)
}

//msgp:tuple MessageBodiesRequest
type MessageBodiesRequest struct {
	SeqHashes Hashes
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
	Hash *Hash
}

//msgp:tuple MessageGetMsg
type MessageGetMsg struct {
	Hash *Hash
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
	RawCampaign *RawCampaign
}

func (m *MessageCampaign) String() string {
	return m.RawCampaign.String()
}

//msgp:tuple MessageTermChange
type MessageTermChange struct {
	RawTermChange *RawTermChange
}

func (m *MessageTermChange) String() string {
	return m.RawTermChange.String()
}

//msgp:tuple MessageTermChangeResponse
type MessageTermChangeResponse struct {
	TermChange *TermChange
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

//msgp:tuple MessageTypeDkgSigSets
type MessageConsensusDkgSigSets struct {
	PkBls     []byte
	PublicKey []byte
	Signature []byte
	TermId    uint64
}

//msgp:MessageConsensusDkgGenesisPublicKey
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
	var buf bytes.Buffer
	panicIfError(binary.Write(&buf, binary.BigEndian, m.PkBls))
	return buf.Bytes()
}

func (m *MessageConsensusDkgSigSets) String() string {
	return "dkgSigsets" + fmt.Sprintf("len %d", len(m.PkBls))
}

func (m *MessageConsensusDkgDeal) SignatureTargets() []byte {
	var buf bytes.Buffer
	d := []byte(m.Data)
	panicIfError(binary.Write(&buf, binary.BigEndian, d))
	panicIfError(binary.Write(&buf, binary.BigEndian, m.Id))

	return buf.Bytes()
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
	var buf bytes.Buffer
	d := []byte(m.Data)
	panicIfError(binary.Write(&buf, binary.BigEndian, d))
	panicIfError(binary.Write(&buf, binary.BigEndian, m.Id))

	return buf.Bytes()
}
