package message

import (
	"fmt"
	"strconv"
)

//go:generate msgp

type OgMessageType int

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

func (o OgMessageType) String() string {
	switch o {
	case OgMessageTypePing:
		return "OgMessageTypePing"
	case OgMessageTypePong:
		return "OgMessageTypePong"
	default:
		return "Unknown Message " + strconv.Itoa(int(o))
	}
}

//msgp OgMessagePing
type OgMessagePing struct {
	Protocol string
}

func (z *OgMessagePing) GetType() int {
	return int(OgMessageTypePing)
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

//msgp OgMessagePong
type OgMessagePong struct {
	Protocol string
	Close    bool
}

func (m *OgMessagePong) String() string {
	return "MessageTypePong"
}

func (m *OgMessagePong) GetType() int {
	return int(OgMessageTypePong)
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

////msgp OgMessageBatchSyncRequest
//type OgMessageBatchSyncRequest struct {
//	Hashes [][]byte
//	//BloomFilter []byte
//	RequestId uint32 //avoid message drop
//	//HashTerminats *HashTerminats
//}
//
//func (m *OgMessageBatchSyncRequest) GetType() int {
//	return int(OgMessageTypeBatchSyncRequest)
//}
//
//func (m *OgMessageBatchSyncRequest) ToBytes() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *OgMessageBatchSyncRequest) String() string {
//	return fmt.Sprintf("OgMessageBatchSyncRequest[req %d]", m.RequestId)
//}
//
//func (z *OgMessageBatchSyncRequest) FromBytes(bts []byte) error {
//	_, err := z.UnmarshalMsg(bts)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
////msgp OgMessageSyncResponse
//type OgMessageSyncResponse struct {
//	//RawTxs *RawTxs
//	////SequencerIndex  []uint32
//	//RawSequencers  *RawSequencers
//	//RawCampaigns   *RawCampaigns
//	//RawTermChanges *RawTermChanges
//	//RawTxs    *TxisMarshaler
//	RequestId uint32 //avoid message drop
//	Height    uint64
//	Offset    int
//	Resources []MessageContentResource
//}
//
//func (m *OgMessageSyncResponse) GetType() int {
//	return int(OgMessageTypeSyncResponse)
//}
//
//func (m *OgMessageSyncResponse) ToBytes() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *OgMessageSyncResponse) String() string {
//	return fmt.Sprintf("OgMessageSyncResponse[req %d height %d]", m.RequestId, len(m.Resources))
//}
//
//func (z *OgMessageSyncResponse) FromBytes(bts []byte) error {
//	_, err := z.UnmarshalMsg(bts)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
////msgp OgMessageQueryStatusRequest
//type OgMessageQueryStatusRequest struct{}
//
//func (m *OgMessageQueryStatusRequest) GetType() int {
//	return int(OgMessageTypeQueryStatusRequest)
//}
//
//func (m *OgMessageQueryStatusRequest) ToBytes() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *OgMessageQueryStatusRequest) String() string {
//	return "OgMessageQueryStatusRequest"
//}
//
//func (m *OgMessageQueryStatusRequest) FromBytes(bts []byte) error {
//	_, err := m.UnmarshalMsg(bts)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
////msgp OgMessageQueryStatusResponse
//type OgMessageQueryStatusResponse struct {
//	ProtocolVersion uint32
//	NetworkId       uint64
//	CurrentBlock    types.Hash
//	GenesisBlock    types.Hash
//	CurrentHeight   uint64
//}
//
//func (m *OgMessageQueryStatusResponse) GetType() int {
//	return int(OgMessageTypeQueryStatusResponse)
//}
//
//func (m *OgMessageQueryStatusResponse) ToBytes() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (m *OgMessageQueryStatusResponse) String() string {
//	return "OgMessageQueryStatusResponse"
//}
//
//func (m *OgMessageQueryStatusResponse) FromBytes(bts []byte) error {
//	_, err := m.UnmarshalMsg(bts)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
////msgp OgMessageNewResource
//type OgMessageNewResource struct {
//	Resources []MessageContentResource
//}
//
//func (m *OgMessageNewResource) GetType() int {
//	return int(OgMessageTypeNewResource)
//}
//
//func (m *OgMessageNewResource) ToBytes() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//func (m *OgMessageNewResource) String() string {
//	return "OgMessageNewResource"
//}
//
//func (z *OgMessageNewResource) FromBytes(bts []byte) error {
//	_, err := z.UnmarshalMsg(bts)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
////msgp HandleMessageHeightSyncRequest
//type OgMessageHeightSyncRequest struct {
//	//Hashes common.Hashes
//	//SeqHash   common.Hash
//	Height    uint64
//	Offset    int
//	RequestId uint32 //avoid message drop
//}
//
//func (z *OgMessageHeightSyncRequest) GetType() int {
//	return int(OgMessageTypeHeightSyncRequest)
//}
//
//func (m *OgMessageHeightSyncRequest) ToBytes() []byte {
//	b, err := m.MarshalMsg(nil)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}
//
//func (z *OgMessageHeightSyncRequest) FromBytes(bts []byte) error {
//	_, err := z.UnmarshalMsg(bts)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (m *OgMessageHeightSyncRequest) String() string {
//	return fmt.Sprintf("height: %d, offset: %d, requestId: %d", m.Height, m.Offset, m.RequestId)
//}

////msgp MessageTxsResponse
//type MessageTxsResponse struct {
//	//RawTxs         *RawTxs
//	//RawSequencer *RawSequencer
//	//RawCampaigns   *RawCampaigns
//	//RawTermChanges *RawTermChanges
//	//RawTxs      *TxisMarshaler
//	RequestedId uint32 //avoid message drop
//	Resources   []MessageContentResource
//}
//
//func (m *MessageTxsResponse) GetType() int {
//	return int(OgMessageType(MessageTypeTxsResponse)
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

//msgp OgMessageHeaderRequest
type OgMessageHeaderRequest struct {
	//Origin    HashOrNumber // Block from which to retrieve headers
	Amount    uint64 // Maximum number of headers to retrieve
	Skip      uint64 // Blocks to skip between consecutive headers
	Reverse   bool   // Query direction (false = rising towards latest, true = falling towards genesis)
	RequestId uint32 //avoid message drop
}

func (m *OgMessageHeaderRequest) GetType() int {
	return int(OgMessageTypeHeaderRequest)
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

////msgp MessageSequencerHeader
//type MessageSequencerHeader struct {
//	Hash   common.Hash
//	Number uint64
//}
//
//func (m *MessageSequencerHeader) GetType() int {
//	return int(OgMessageType(MessageTypeSequencerHeader)
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
////msgp MessageHeaderResponse
//type MessageHeaderResponse struct {
//	Headers     *SequencerHeaders
//	RequestedId uint32 //avoid message drop
//}
//
//func (m *MessageHeaderResponse) GetType() int {
//	return int(OgMessageType(MessageTypeHeaderResponse)
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
////msgp MessageBodiesRequest
//type MessageBodiesRequest struct {
//	SeqHashes common.Hashes
//	RequestId uint32 //avoid message drop
//}
//
//func (m *MessageBodiesRequest) GetType() int {
//	return int(OgMessageType(MessageTypeBodiesRequest)
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
////msgp MessageBodiesResponse
//type MessageBodiesResponse struct {
//	Bodies      []RawData
//	RequestedId uint32 //avoid message drop
//}
//
//func (m *MessageBodiesResponse) GetType() int {
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

////msgp MessageControl
//type MessageControl struct {
//	Hash *common.Hash
//}
//
//func (m *MessageControl) GetType() int {
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

////msgp MessageGetMsg
//type MessageGetMsg struct {
//	Hash *common.Hash
//}
//
//func (m *MessageGetMsg) GetType() int {
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

////msgp MessageGetMsg
//type MessageDuplicate bool
//
//func (m *MessageDuplicate) GetType() int {
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
////msgp MessageNewActionTx
//type MessageNewActionTx struct {
//	ActionTx *ActionTx
//}
//
//func (m *MessageNewActionTx) GetType() int {
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
