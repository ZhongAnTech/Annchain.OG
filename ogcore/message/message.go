package message

import (
	"fmt"
	"github.com/annchain/OG/common"
)

//go:generate msgp

type OgMessageType uint16

// og protocol message codes
// TODO: use MessageTypeManager to manage global messages
// basic messages ids range from [0, 100)
// bft consensus: [100, 200)
// dkg: [200, 300)
// campaign: [300, 400)
const (
	OgMessageTypeStatus OgMessageType = iota + 0
	OgMessageTypePing
	OgMessageTypePong
	OgMessageTypeBatchSyncRequest
	OgMessageTypeSyncResponse
	MessageTypeFetchByHashRequest
	MessageTypeFetchByHashResponse
	OgMessageTypeQueryStatusRequest
	OgMessageTypeQueryStatusResponse
	OgMessageTypeNewResource
	OgMessageTypeHeightSyncRequest

	//MessageTypeNewSequencer
	//MessageTypeNewTxs
	MessageTypeSequencerHeader

	MessageTypeBodiesRequest
	MessageTypeBodiesResponse

	OgMessageTypeTxsRequest
	MessageTypeTxsResponse
	OgMessageTypeHeaderRequest
	MessageTypeHeaderResponse

	//for optimizing network
	MessageTypeGetMsg
	MessageTypeDuplicate
	MessageTypeControl

	//move to campaign
	//MessageTypeCampaign
	//MessageTypeTermChange

	MessageTypeArchive
	MessageTypeActionTX

	//move to dkg package
	//MessageTypeConsensusDkgDeal
	//MessageTypeConsensusDkgDealResponse
	//MessageTypeConsensusDkgSigSets
	//MessageTypeConsensusDkgGenesisPublicKey

	//move to campaign
	//MessageTypeTermChangeRequest
	//MessageTypeTermChangeResponse

	MessageTypeSecret //encrypted message

	// move to bft package
	//MessageTypeProposal
	//MessageTypePreVote
	//MessageTypePreCommit

	MessageTypeOg01Length //og01 length

	// Protocol messages belonging to og/02

	GetNodeDataMsg
	NodeDataMsg
	GetReceiptsMsg
	MessageTypeOg02Length

	MessageTypeNewArchive
	MessageTypeNewActionTx
)

type OgMessage interface {
	Bytable
	GetType() OgMessageType
	String() string
}

//msgp:tuple OgMessagePing
type OgMessagePing struct{}

func (z *OgMessagePing) GetType() OgMessageType {
	return OgMessageTypePing
}

func (m *OgMessagePing) ToBytes() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgMessagePing) String() string {
	return "MessageTypePing"
}

func (z *OgMessagePing) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple OgMessagePong
type OgMessagePong struct{}

func (m *OgMessagePong) String() string {
	return "MessageTypePong"
}

func (m *OgMessagePong) GetType() OgMessageType {
	return OgMessageTypePong
}

func (m *OgMessagePong) ToBytes() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgMessagePong) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple OgMessageBatchSyncRequest
type OgMessageBatchSyncRequest struct {
	Height      *uint64
	Hashes      common.Hashes
	BloomFilter []byte
	RequestId   uint32 //avoid msg drop
	//HashTerminats *HashTerminats
}

func (m *OgMessageBatchSyncRequest) GetType() OgMessageType {
	return OgMessageTypeBatchSyncRequest
}

func (m *OgMessageBatchSyncRequest) ToBytes() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *OgMessageBatchSyncRequest) String() string {
	return fmt.Sprintf("OgMessageBatchSyncRequest[req %d]", m.RequestId)
}

func (z *OgMessageBatchSyncRequest) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple OgMessageSyncResponse
type OgMessageSyncResponse struct {
	//RawTxs *RawTxs
	////SequencerIndex  []uint32
	//RawSequencers  *RawSequencers
	//RawCampaigns   *RawCampaigns
	//RawTermChanges *RawTermChanges
	//RawTxs    *TxisMarshaler
	RequestId uint32 //avoid msg drop
	Height    uint64
	Offset    uint32
	Resources []MessageContentResource
}

func (m *OgMessageSyncResponse) GetType() OgMessageType {
	return OgMessageTypeSyncResponse
}

func (m *OgMessageSyncResponse) ToBytes() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *OgMessageSyncResponse) String() string {
	return fmt.Sprintf("OgMessageSyncResponse[req %d height %d]", m.RequestId, len(m.Resources))
}

func (z *OgMessageSyncResponse) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple OgMessageQueryStatusRequest
type OgMessageQueryStatusRequest struct{}

func (m *OgMessageQueryStatusRequest) GetType() OgMessageType {
	return OgMessageTypeQueryStatusRequest
}

func (m *OgMessageQueryStatusRequest) ToBytes() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *OgMessageQueryStatusRequest) String() string {
	return "OgMessageQueryStatusRequest"
}

func (m *OgMessageQueryStatusRequest) FromBytes(bts []byte) error {
	_, err := m.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple OgMessageQueryStatusResponse
type OgMessageQueryStatusResponse struct {
	ProtocolVersion uint32
	NetworkId       uint64
	CurrentBlock    common.Hash
	GenesisBlock    common.Hash
	CurrentHeight   uint64
}

func (m *OgMessageQueryStatusResponse) GetType() OgMessageType {
	return OgMessageTypeQueryStatusResponse
}

func (m *OgMessageQueryStatusResponse) ToBytes() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *OgMessageQueryStatusResponse) String() string {
	return "OgMessageQueryStatusResponse"
}

func (m *OgMessageQueryStatusResponse) FromBytes(bts []byte) error {
	_, err := m.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple OgMessageNewResource
type OgMessageNewResource struct {
	Resources []MessageContentResource
}

func (m *OgMessageNewResource) GetType() OgMessageType {
	return OgMessageTypeNewResource
}

func (m *OgMessageNewResource) ToBytes() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}
func (m *OgMessageNewResource) String() string {
	return "OgMessageNewResource"
}

func (z *OgMessageNewResource) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp:tuple HandleMessageHeightSyncRequest
type OgMessageHeightSyncRequest struct {
	//Hashes common.Hashes
	//SeqHash   common.Hash
	Height    uint64
	Offset    uint32
	RequestId uint32 //avoid msg drop
}

func (z *OgMessageHeightSyncRequest) GetType() OgMessageType {
	return OgMessageTypeHeightSyncRequest
}

func (m *OgMessageHeightSyncRequest) ToBytes() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgMessageHeightSyncRequest) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (m *OgMessageHeightSyncRequest) String() string {
	return fmt.Sprintf("height: %d, offset: %d, requestId: %d", m.Height, m.Offset, m.RequestId)
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
//func (m *MessageTxsResponse) GetType() OgMessageType {
//	return OgMessageType(MessageTypeTxsResponse)
//}
//
//func (m *MessageTxsResponse) GetBytes() []byte {
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
//		Data: m.GetBytes(),
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

//msgp:tuple OgMessageHeaderRequest
type OgMessageHeaderRequest struct {
	//Origin    HashOrNumber // Block from which to retrieve headers
	Amount    uint64 // Maximum number of headers to retrieve
	Skip      uint64 // Blocks to skip between consecutive headers
	Reverse   bool   // Query direction (false = rising towards latest, true = falling towards genesis)
	RequestId uint32 //avoid msg drop
}

func (m *OgMessageHeaderRequest) GetType() OgMessageType {
	return OgMessageTypeHeaderRequest
}

func (m *OgMessageHeaderRequest) ToBytes() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *OgMessageHeaderRequest) String() string {
	return fmt.Sprintf("OgMessageHeaderRequest amount : %d ,skip : %d, reverse : %v, requestId :%d", m.Amount, m.Skip, m.Reverse, m.RequestId)
}

func (z *OgMessageHeaderRequest) Marshal() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgMessageHeaderRequest) FromBytes(bts []byte) error {
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
//func (m *MessageSequencerHeader) GetType() OgMessageType {
//	return OgMessageType(MessageTypeSequencerHeader)
//}
//
//func (m *MessageSequencerHeader) GetBytes() []byte {
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
//		Data: m.GetBytes(),
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
//func (m *MessageHeaderResponse) GetType() OgMessageType {
//	return OgMessageType(MessageTypeHeaderResponse)
//}
//
//func (m *MessageHeaderResponse) GetBytes() []byte {
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
//		Data: m.GetBytes(),
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
//func (m *MessageBodiesRequest) GetType() OgMessageType {
//	return OgMessageType(MessageTypeBodiesRequest)
//}
//
//func (m *MessageBodiesRequest) GetBytes() []byte {
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
//		Data: m.GetBytes(),
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
//func (m *MessageBodiesResponse) GetType() OgMessageType {
//	return MessageTypeBodiesResponse
//}
//
//func (m *MessageBodiesResponse) GetBytes() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageBodiesResponse) ToBinary() []byte {
//	return []byte{
//		Type: OgMessageType(m.GetType()),
//		Data: m.GetBytes(),
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
//func (m *MessageControl) GetType() OgMessageType {
//	return MessageTypeControl
//}
//
//func (m *MessageControl) GetBytes() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageControl) ToBinary() []byte {
//	return []byte{
//		Type: OgMessageType(m.GetType()),
//		Data: m.GetBytes(),
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
//func (m *MessageGetMsg) GetType() OgMessageType {
//	return MessageTypeGetMsg
//}
//
//func (m *MessageGetMsg) GetBytes() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageGetMsg) ToBinary() []byte {
//	return []byte{
//		Type: OgMessageType(m.GetType()),
//		Data: m.GetBytes(),
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
//func (m *MessageDuplicate) GetType() OgMessageType {
//	return MessageTypeDuplicate
//}
//
//func (m *MessageDuplicate) GetBytes() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageDuplicate) ToBinary() []byte {
//	return []byte{
//		Type: OgMessageType(m.GetType()),
//		Data: m.GetBytes(),
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
//func (m *MessageNewActionTx) GetType() OgMessageType {
//	return MessageTypeNewActionTx
//}
//
//func (m *MessageNewActionTx) GetBytes() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *MessageNewActionTx) ToBinary() []byte {
//	return []byte{
//		Type: OgMessageType(m.GetType()),
//		Data: m.GetBytes(),
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
