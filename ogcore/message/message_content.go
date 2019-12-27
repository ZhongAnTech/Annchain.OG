package message

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
)

type ResourceType uint8

const (
	ResourceTypeTx ResourceType = iota
	ResourceTypeSequencer
	ResourceTypeArchive
	ResourceTypeAction
)

var ResourceTypeStrings = map[ResourceType]string{
	ResourceTypeTx:        "RTTx",
	ResourceTypeSequencer: "RTSq",
}

//go:generate msgp

//msgp:tuple MessageContentResource
type MessageContentResource struct {
	ResourceType    ResourceType
	ResourceContent []byte
}

func (z *MessageContentResource) String() string {
	return fmt.Sprintf("%s %d %s", ResourceTypeStrings[z.ResourceType], len(z.ResourceContent), hexutil.Encode(z.ResourceContent))
}

//msgp:tuple MessageContentTx
type MessageContentTx struct {
	Hash         common.Hash
	ParentsHash  []common.Hash
	MineNonce    uint64
	AccountNonce uint64
	From         common.Address
	To           common.Address
	Value        *math.BigInt
	TokenId      int32
	PublicKey    []byte
	Data         []byte
	Signature    []byte
}

func (z *MessageContentTx) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessageContentTx) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple MessageContentSequencer
type MessageContentSequencer struct {
	Hash         common.Hash
	ParentsHash  []common.Hash
	MineNonce    uint64
	AccountNonce uint64
	Issuer       common.Address
	PublicKey    []byte
	Signature    []byte
	StateRoot    common.Hash
	Height       uint64
}

func (z *MessageContentSequencer) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessageContentSequencer) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}
