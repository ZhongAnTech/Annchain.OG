package og

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/types/msg"
)

//go:generate msgp

//msgp:tuple MessagePing
type MessagePing struct{}

func (z *MessagePing) GetType() msg.OgMessageType {
	return MessageTypePing
}

func (z *MessagePing) String() string {
	return "MessageTypePing"
}

func (z *MessagePing) Marshal() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessagePing) Unmarshal(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple MessagePong
type MessagePong struct{}

func (m *MessagePong) String() string {
	return "MessageTypePong"
}

func (m *MessagePong) GetType() msg.OgMessageType {
	return MessageTypePong
}

func (m *MessagePong) GetData() []byte {
	return nil
}

func (z *MessagePong) Marshal() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessagePong) Unmarshal(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple MessageBatchSyncRequest
type MessageBatchSyncRequest struct {
	Hashes      common.Hashes
	BloomFilter []byte
	RequestId   uint32 //avoid msg drop

	//HashTerminats *HashTerminats
	//Height      *uint64
}

func (m *MessageBatchSyncRequest) GetType() msg.OgMessageType {
	return MessageTypeBatchSyncRequest
}

func (m *MessageBatchSyncRequest) String() string {
	return fmt.Sprintf("MessageBatchSyncRequest[req %d]", m.RequestId)
}

func (z *MessageBatchSyncRequest) Marshal() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessageBatchSyncRequest) Unmarshal(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
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

func (m *MessageSyncResponse) GetType() msg.OgMessageType {
	return MessageTypeSyncResponse
}

func (m *MessageSyncResponse) String() string {
	return fmt.Sprintf("MessageSyncResponse[req %d height %d]", m.RequestId, len(m.Resources))
}

func (z *MessageSyncResponse) Marshal() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessageSyncResponse) Unmarshal(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple MessageNewResource
type MessageNewResource struct {
	Resources []MessageContentResource
}

func (m *MessageNewResource) GetType() msg.OgMessageType {
	return MessageTypeNewResource
}

func (m *MessageNewResource) String() string {
	return "MessageNewResource"
}

func (z *MessageNewResource) Marshal() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessageNewResource) Unmarshal(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple MessageHeightSyncRequest
type MessageHeightSyncRequest struct {
	//Hashes common.Hashes
	//SeqHash   common.Hash
	Height    uint64
	Id        uint64
	RequestId uint32 //avoid msg drop
}

func (z *MessageHeightSyncRequest) GetType() msg.OgMessageType {
	return MessageTypeTxsRequest
}

func (m *MessageHeightSyncRequest) String() string {
	return fmt.Sprintf("height: %d, id : %d, requestId : %d", m.Height, m.Id, m.RequestId)
}

func (z *MessageHeightSyncRequest) Marshal() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessageHeightSyncRequest) Unmarshal(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}


////msgp:tuple MessageTxsResponse
//type MessageTxsResponse struct {
//	//RawTxs         *RawTxs
//	//RawSequencer *RawSequencer
//	//RawCampaigns   *RawCampaigns
//	//RawTermChanges *RawTermChanges
//	//RawTxs      *TxisMarshaler
//	RequestedId uint32 //avoid msg drop
//	Resources   []MessageContentResource
//}
//
//func (m *MessageTxsResponse) GetType() msg.OgMessageType {
//	return msg.OgMessageType(MessageTypeTxsResponse)
//}
//
//func (m *MessageTxsResponse) GetData() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageTxsResponse) ToBinary() []byte {
//	return []byte{
//		Type: m.GetType(),
//		Data: m.GetData(),
//	}
//}
//
//func (m *MessageTxsResponse) FromBinary(bs []byte) error {
//	_, err := m.UnmarshalMsg(bs)
//	return err
//}
//
//func (m *MessageTxsResponse) String() string {
//	return fmt.Sprintf("txs: [%s], Sequencer: %s, requestedId %d", m.RawTxs.String(), m.RawSequencer.String(), m.RequestedId)
//}

// getBlockHeadersData represents a block header query.

//msgp:tuple MessageHeaderRequest
type MessageHeaderRequest struct {
	//Origin    HashOrNumber // Block from which to retrieve headers
	Amount    uint64 // Maximum number of headers to retrieve
	Skip      uint64 // Blocks to skip between consecutive headers
	Reverse   bool   // Query direction (false = rising towards latest, true = falling towards genesis)
	RequestId uint32 //avoid msg drop
}

func (m *MessageHeaderRequest) GetType() msg.OgMessageType {
	return MessageTypeHeaderRequest
}

func (m *MessageHeaderRequest) String() string {
	return fmt.Sprintf("MessageHeaderRequest amount : %d ,skip : %d, reverse : %v, requestId :%d", m.Amount, m.Skip, m.Reverse, m.RequestId)
}

func (z *MessageHeaderRequest) Marshal() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessageHeaderRequest) Unmarshal(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

////msgp:tuple MessageSequencerHeader
//type MessageSequencerHeader struct {
//	Hash   common.Hash
//	Number uint64
//}
//
//func (m *MessageSequencerHeader) GetType() msg.OgMessageType {
//	return msg.OgMessageType(MessageTypeSequencerHeader)
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
//func (m *MessageSequencerHeader) ToBinary() []byte {
//	return []byte{
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
//func (m *MessageHeaderResponse) GetType() msg.OgMessageType {
//	return msg.OgMessageType(MessageTypeHeaderResponse)
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
//func (m *MessageHeaderResponse) ToBinary() []byte {
//	return []byte{
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
//func (m *MessageBodiesRequest) GetType() msg.OgMessageType {
//	return msg.OgMessageType(MessageTypeBodiesRequest)
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
//func (m *MessageBodiesRequest) ToBinary() []byte {
//	return []byte{
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
//func (m *MessageBodiesResponse) GetType() msg.OgMessageType {
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
//func (m *MessageBodiesResponse) ToBinary() []byte {
//	return []byte{
//		Type: msg.OgMessageType(m.GetType()),
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
//func (m *MessageControl) GetType() msg.OgMessageType {
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
//func (m *MessageControl) ToBinary() []byte {
//	return []byte{
//		Type: msg.OgMessageType(m.GetType()),
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
//func (m *MessageGetMsg) GetType() msg.OgMessageType {
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
//func (m *MessageGetMsg) ToBinary() []byte {
//	return []byte{
//		Type: msg.OgMessageType(m.GetType()),
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
//func (m *MessageDuplicate) GetType() msg.OgMessageType {
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
//func (m *MessageDuplicate) ToBinary() []byte {
//	return []byte{
//		Type: msg.OgMessageType(m.GetType()),
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
//func (m *MessageNewActionTx) GetType() msg.OgMessageType {
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
//func (m *MessageNewActionTx) ToBinary() []byte {
//	return []byte{
//		Type: msg.OgMessageType(m.GetType()),
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
