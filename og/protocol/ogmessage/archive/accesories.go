package archive

import (
	"fmt"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/bloom"
	"strings"
)

//go:generate msgp

const (
	BloomItemNumber = 3000
	HashFuncNum     = 8
)

//msgp:tuple BloomFilter
type BloomFilter struct {
	Data     []byte
	Count    uint32
	Capacity uint32
	filter   *bloom.BloomFilter
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
	c.Capacity = m
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

type HashTerminat [4]byte

type HashTerminats []HashTerminat

func (h HashTerminat) String() string {
	return hexutil.Encode(h[:])
}

func (h HashTerminats) String() string {
	var strs []string
	for _, v := range h {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

//msgp:tuple MessageBodyData
//type MessageBodyData struct {
//	//RawTxs         *RawTxs
//	//RawTermChanges *RawTermChanges
//	//RawCampaigns   *RawCampaigns
//	RawSequencer *RawSequencer
//	RawTxs       *TxisMarshaler
//}
//
//func (m *MessageBodyData) ToTxis() Txis {
//	var txis Txis
//	if m.RawTxs != nil {
//		txs := m.RawTxs.Txis()
//		txis = append(txis, txs...)
//	}
//	if len(txis) == 0 {
//		return nil
//	}
//	return txis
//}
//
//func (m *MessageBodyData) String() string {
//	return fmt.Sprintf("txs: [%s], Sequencer: %s", m.RawTxs.String(), m.RawSequencer.String())
//}

// hashOrNumber is a combined field for specifying an origin block.
//msgp:tuple HashOrNumber
type HashOrNumber struct {
	Hash   *types.Hash // Block hash from which to retrieve headers (excludes Number)
	Number *uint64     // Block hash from which to retrieve headers (excludes Hash)
}

func (m *HashOrNumber) String() string {
	if m.Hash == nil {
		return fmt.Sprintf("hash: nil, number : %d ", *m.Number)
	}
	return fmt.Sprintf("hash: %s, number : %d", m.Hash.String(), m.Number)
}

//msgp:tuple RawData
type RawData []byte
