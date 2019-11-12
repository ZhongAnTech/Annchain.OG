package ogmessage

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/types/msg"
)

//go:generate msgp

//msgp:tuple MessagePing
type MessagePing struct{}

func (z MessagePing) String() string {
	return "MessageTypePing"
}

func (m *MessagePing) GetType() msg.BinaryMessageType {
	return msg.BinaryMessageType(MessageTypePing)
}

func (m *MessagePing) GetData() []byte {
	return []byte{}
}

func (m *MessagePing) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (z *MessagePing) FromBinary([]byte) error {
	// do nothing since the array is always empty
	return nil
}

//msgp:tuple MessagePong
type MessagePong struct{}

func (m *MessagePong) String() string {
	return "MessageTypePong"
}

func (m *MessagePong) GetType() msg.BinaryMessageType {
	return msg.BinaryMessageType(MessageTypePong)
}

func (m *MessagePong) GetData() []byte {
	return []byte{}
}

func (m *MessagePong) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: msg.BinaryMessageType(m.GetType()),
		Data: m.GetData(),
	}
}

func (z *MessagePong) FromBinary([]byte) error {
	// do nothing since the array is always empty
	return nil
}

//msgp:tuple MessageSyncRequest
type MessageSyncRequest struct {
	Hashes      common.Hashes
	BloomFilter []byte
	RequestId   uint32 //avoid msg drop

	//HashTerminats *HashTerminats
	//Height      *uint64
}

func (m *MessageSyncRequest) GetType() msg.BinaryMessageType {
	return msg.BinaryMessageType(MessageTypeSyncRequest)
}

func (m *MessageSyncRequest) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageSyncRequest) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageSyncRequest) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageSyncRequest) String() string {
	return fmt.Sprintf("MessageSyncRequest[req %d]", m.RequestId)
}

//msgp:tuple MessageSyncResponse
type MessageSyncResponse struct {
	//RawTxs *RawTxs
	////SequencerIndex  []uint32
	//RawSequencers  *RawSequencers
	//RawCampaigns   *RawCampaigns
	//RawTermChanges *RawTermChanges
	//RawTxs    *TxisMarshaler
	RequestId uint32 //avoid msg drop
	Resources []MessageContentResource
}

func (m *MessageSyncResponse) GetType() msg.BinaryMessageType {
	return msg.BinaryMessageType(MessageTypeSyncResponse)
}

func (m *MessageSyncResponse) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageSyncResponse) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageSyncResponse) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageSyncResponse) String() string {
	return fmt.Sprintf("MessageSyncResponse[req %d height %d]", m.RequestId, len(m.Resources))
}

type MessageNewResource struct {
	Resources []MessageContentResource
}

func (m MessageNewResource) GetType() msg.BinaryMessageType {
	return msg.BinaryMessageType(MessageTypeNewResource)
}

func (m MessageNewResource) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m MessageNewResource) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m MessageNewResource) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m MessageNewResource) String() string {
	return "MessageNewResource"
}

//msgp:tuple MessageTxsRequest
type MessageTxsRequest struct {
	Hashes    common.Hashes
	SeqHash   common.Hash
	Id        uint64
	RequestId uint32 //avoid msg drop
}

func (z *MessageTxsRequest) GetType() msg.BinaryMessageType {
	return msg.BinaryMessageType(MessageTypeTxsRequest)
}

func (m *MessageTxsRequest) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageTxsRequest) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageTxsRequest) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageTxsRequest) String() string {
	return fmt.Sprintf("hashes: [%s], seqHash: %s, id : %d, requstId : %d", m.Hashes.String(), m.SeqHash.String(), m.Id, m.RequestId)
}

//msgp:tuple MessageTxsResponse
type MessageTxsResponse struct {
	//RawTxs         *RawTxs
	//RawSequencer *RawSequencer
	//RawCampaigns   *RawCampaigns
	//RawTermChanges *RawTermChanges
	//RawTxs      *TxisMarshaler
	RequestedId uint32 //avoid msg drop
	Resources   []MessageContentResource
}

func (m *MessageTxsResponse) GetType() msg.BinaryMessageType {
	return msg.BinaryMessageType(MessageTypeTxsResponse)
}

func (m *MessageTxsResponse) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageTxsResponse) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageTxsResponse) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageTxsResponse) String() string {
	return fmt.Sprintf("txs: [%s], Sequencer: %s, requestedId %d", m.RawTxs.String(), m.RawSequencer.String(), m.RequestedId)
}

// getBlockHeadersData represents a block header query.
//msgp:tuple MessageHeaderRequest
type MessageHeaderRequest struct {
	//Origin    HashOrNumber // Block from which to retrieve headers
	Amount    uint64       // Maximum number of headers to retrieve
	Skip      uint64       // Blocks to skip between consecutive headers
	Reverse   bool         // Query direction (false = rising towards latest, true = falling towards genesis)
	RequestId uint32       //avoid msg drop
}

func (m *MessageHeaderRequest) GetType() msg.BinaryMessageType {
	return msg.BinaryMessageType(MessageTypeHeaderRequest)
}

func (m *MessageHeaderRequest) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageHeaderRequest) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageHeaderRequest) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageHeaderRequest) String() string {
	return fmt.Sprintf("MessageHeaderRequest amount : %d ,skip : %d, reverse : %v, requestId :%d", m.Amount, m.Skip, m.Reverse, m.RequestId)

}

////msgp:tuple MessageSequencerHeader
//type MessageSequencerHeader struct {
//	Hash   common.Hash
//	Number uint64
//}
//
//func (m *MessageSequencerHeader) GetType() msg.BinaryMessageType {
//	return msg.BinaryMessageType(MessageTypeSequencerHeader)
//}
//
//func (m *MessageSequencerHeader) GetData() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageSequencerHeader) ToBinary() msg.BinaryMessage {
//	return msg.BinaryMessage{
//		Type: m.GetType(),
//		Data: m.GetData(),
//	}
//}
//
//func (m *MessageSequencerHeader) FromBinary(bs []byte) error {
//	_, err := m.UnmarshalMsg(bs)
//	return err
//}
//
//func (m *MessageSequencerHeader) String() string {
//	return fmt.Sprintf("hash: %s, number : %d", m.Hash.String(), m.Number)
//}
//
////msgp:tuple MessageHeaderResponse
//type MessageHeaderResponse struct {
//	Headers     *SequencerHeaders
//	RequestedId uint32 //avoid msg drop
//}
//
//func (m *MessageHeaderResponse) GetType() msg.BinaryMessageType {
//	return msg.BinaryMessageType(MessageTypeHeaderResponse)
//}
//
//func (m *MessageHeaderResponse) GetData() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageHeaderResponse) ToBinary() msg.BinaryMessage {
//	return msg.BinaryMessage{
//		Type: m.GetType(),
//		Data: m.GetData(),
//	}
//}
//
//func (m *MessageHeaderResponse) FromBinary(bs []byte) error {
//	_, err := m.UnmarshalMsg(bs)
//	return err
//}
//
//func (m *MessageHeaderResponse) String() string {
//	return fmt.Sprintf("headers: [%s] reuqestedId :%d", m.Headers.String(), m.RequestedId)
//}
//
////msgp:tuple MessageBodiesRequest
//type MessageBodiesRequest struct {
//	SeqHashes common.Hashes
//	RequestId uint32 //avoid msg drop
//}
//
//func (m *MessageBodiesRequest) GetType() msg.BinaryMessageType {
//	return msg.BinaryMessageType(MessageTypeBodiesRequest)
//}
//
//func (m *MessageBodiesRequest) GetData() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageBodiesRequest) ToBinary() msg.BinaryMessage {
//	return msg.BinaryMessage{
//		Type: m.GetType(),
//		Data: m.GetData(),
//	}
//}
//
//func (m *MessageBodiesRequest) FromBinary(bs []byte) error {
//	_, err := m.UnmarshalMsg(bs)
//	return err
//}
//
//func (m *MessageBodiesRequest) String() string {
//	return m.SeqHashes.String() + fmt.Sprintf(" requestId :%d", m.RequestId)
//}
//
////msgp:tuple MessageBodiesResponse
//type MessageBodiesResponse struct {
//	Bodies      []RawData
//	RequestedId uint32 //avoid msg drop
//}
//
//func (m *MessageBodiesResponse) GetType() msg.BinaryMessageType {
//	return MessageTypeBodiesResponse
//}
//
//func (m *MessageBodiesResponse) GetData() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageBodiesResponse) ToBinary() msg.BinaryMessage {
//	return msg.BinaryMessage{
//		Type: msg.BinaryMessageType(m.GetType()),
//		Data: m.GetData(),
//	}
//}
//
//func (m *MessageBodiesResponse) FromBinary(bs []byte) error {
//	_, err := m.UnmarshalMsg(bs)
//	return err
//}
//
//func (m *MessageBodiesResponse) String() string {
//	return fmt.Sprintf("bodies len : %d, reuqestedId :%d", len(m.Bodies), m.RequestedId)
//}

////msgp:tuple MessageControl
//type MessageControl struct {
//	Hash *common.Hash
//}
//
//func (m *MessageControl) GetType() msg.BinaryMessageType {
//	return MessageTypeControl
//}
//
//func (m *MessageControl) GetData() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageControl) ToBinary() msg.BinaryMessage {
//	return msg.BinaryMessage{
//		Type: msg.BinaryMessageType(m.GetType()),
//		Data: m.GetData(),
//	}
//}
//
//func (m *MessageControl) FromBinary(bs []byte) error {
//	_, err := m.UnmarshalMsg(bs)
//	return err
//}
//
//func (m *MessageControl) String() string {
//	if m == nil || m.Hash == nil {
//		return ""
//	}
//	return m.Hash.String()
//}

////msgp:tuple MessageGetMsg
//type MessageGetMsg struct {
//	Hash *common.Hash
//}
//
//func (m *MessageGetMsg) GetType() msg.BinaryMessageType {
//	return MessageTypeGetMsg
//}
//
//func (m *MessageGetMsg) GetData() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageGetMsg) ToBinary() msg.BinaryMessage {
//	return msg.BinaryMessage{
//		Type: msg.BinaryMessageType(m.GetType()),
//		Data: m.GetData(),
//	}
//}
//
//func (m *MessageGetMsg) FromBinary(bs []byte) error {
//	_, err := m.UnmarshalMsg(bs)
//	return err
//}
//
//func (m *MessageGetMsg) String() string {
//	if m == nil || m.Hash == nil {
//		return ""
//	}
//	return m.Hash.String()
//}

////msgp:tuple MessageGetMsg
//type MessageDuplicate bool
//
//func (m *MessageDuplicate) GetType() msg.BinaryMessageType {
//	return MessageTypeDuplicate
//}
//
//func (m *MessageDuplicate) GetData() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageDuplicate) ToBinary() msg.BinaryMessage {
//	return msg.BinaryMessage{
//		Type: msg.BinaryMessageType(m.GetType()),
//		Data: m.GetData(),
//	}
//}
//
//func (m *MessageDuplicate) FromBinary(bs []byte) error {
//	_, err := m.UnmarshalMsg(bs)
//	return err
//}
//
//func (m *MessageDuplicate) String() string {
//	return "MessageDuplicate"
//}
//
////msgp:tuple MessageNewActionTx
//type MessageNewActionTx struct {
//	ActionTx *ActionTx
//}
//
//func (m *MessageNewActionTx) GetType() msg.BinaryMessageType {
//	return MessageTypeNewActionTx
//}
//
//func (m *MessageNewActionTx) GetData() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageNewActionTx) ToBinary() msg.BinaryMessage {
//	return msg.BinaryMessage{
//		Type: msg.BinaryMessageType(m.GetType()),
//		Data: m.GetData(),
//	}
//}
//
//func (m *MessageNewActionTx) FromBinary(bs []byte) error {
//	_, err := m.UnmarshalMsg(bs)
//	return err
//}
//
//func (m *MessageNewActionTx) String() string {
//	if m.ActionTx == nil {
//		return "nil"
//	}
//	return m.ActionTx.String()
//}

//msgp:tuple MessageAnnsensus
type MessageAnnsensus struct {
	InnerMessageType annsensus.AnnsensusMessageType
	InnerMessage     []byte
}

func (m MessageAnnsensus) GetType() msg.BinaryMessageType {
	return MessageTypeAnnsensus
}

func (m MessageAnnsensus) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m MessageAnnsensus) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: msg.BinaryMessageType(m.GetType()),
		Data: m.GetData(),
	}
}

func (m MessageAnnsensus) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m MessageAnnsensus) String() string {
	return "MessageAnnsensus"
}
