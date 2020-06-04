package message

//
//import (
//	"fmt"
//	"github.com/annchain/OG/arefactor/og/types"
//	"github.com/annchain/OG/common"
//	"github.com/annchain/OG/common/hexutil"
//	"github.com/annchain/OG/common/math"
//)
//
//type ResourceType uint8
//
//const (
//	ResourceTypeTx ResourceType = iota
//	ResourceTypeSequencer
//	ResourceTypeArchive
//	ResourceTypeAction
//)
//
//var ResourceTypeStrings = map[ResourceType]string{
//	ResourceTypeTx:        "RTTx",
//	ResourceTypeSequencer: "RTSq",
//}
//
////go:generate msgp
//
////msgp MessageContentResource
//type MessageContentResource struct {
//	ResourceType    ResourceType
//	ResourceContent []byte
//}
//
//func (z *MessageContentResource) String() string {
//	return fmt.Sprintf("%s %d %s", ResourceTypeStrings[z.ResourceType], len(z.ResourceContent), hexutil.Encode(z.ResourceContent))
//}
//
////msgp MessageContentTx
//type MessageContentTx struct {
//	Hash         types.Hash
//	ParentsHash  []types.Hash
//	MineNonce    uint64
//	AccountNonce uint64
//	From         common.Address
//	To           common.Address
//	Value        *math.BigInt
//	TokenId      int32
//	PublicKey    []byte
//	Data         []byte
//	Signature    []byte
//}
//
//func (z *MessageContentTx) ToBytes() []byte {
//	b, err := z.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (z *MessageContentTx) FromBytes(bts []byte) error {
//	_, err := z.UnmarshalMsg(bts)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
////msgp MessageContentSequencer
//type MessageContentSequencer struct {
//	Hash         types.Hash
//	ParentsHash  []types.Hash
//	MineNonce    uint64
//	AccountNonce uint64
//	Issuer       common.Address
//	PublicKey    []byte
//	Signature    []byte
//	StateRoot    types.Hash
//	Height       uint64
//}
//
//func (z *MessageContentSequencer) ToBytes() []byte {
//	b, err := z.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (z *MessageContentSequencer) FromBytes(bts []byte) error {
//	_, err := z.UnmarshalMsg(bts)
//	if err != nil {
//		return err
//	}
//	return nil
//}
