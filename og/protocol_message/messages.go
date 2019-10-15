package protocol_message

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/types/msg"
)

//go:generate msgp

//msgp:tuple MessagePing
type MessagePing struct{}

func (z MessagePing) String() string {
	return "MessageTypePing"
}

func (m *MessagePing) GetType() msg.BinaryMessageType {
	return MessageTypePing
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
	return MessageTypePong
}

func (m *MessagePong) GetData() []byte {
	return []byte{}
}

func (m *MessagePong) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (z *MessagePong) FromBinary([]byte) error {
	// do nothing since the array is always empty
	return nil
}

//msgp:tuple MessageSyncRequest
type MessageSyncRequest struct {
	Hashes        *common.Hashes
	HashTerminats *HashTerminats
	Filter        *BloomFilter
	Height        *uint64
	RequestId     uint32 //avoid msg drop
}

func (m *MessageSyncRequest) GetType() msg.BinaryMessageType {
	return MessageTypeSyncRequest
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
	var str string
	if m.Filter != nil {
		str = fmt.Sprintf("count: %d", m.Filter.GetCount())
	}
	if m.Hashes != nil {
		str += fmt.Sprintf("hash num %v", m.Hashes.String())
	}
	if m.HashTerminats != nil {
		str += fmt.Sprintf("hashterminates %v ", m.HashTerminats.String())
	}
	str += fmt.Sprintf(" requestId %d  ", m.RequestId)
	return str
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

func (m *MessageSyncResponse) GetType() msg.BinaryMessageType {
	return MessageTypeSyncResponse
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
	//for _,i := range m.SequencerIndex {
	//index = append(index ,fmt.Sprintf("%d",i))
	//}
	return fmt.Sprintf("txs: [%s],requestedId :%d", m.RawTxs.String(), m.RequestedId)
}

//msgp:tuple MessageNewTx
type MessageNewTx struct {
	RawTx *RawTx
}

func (m *MessageNewTx) GetType() msg.BinaryMessageType {
	return MessageTypeNewTx
}

func (m *MessageNewTx) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageNewTx) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageNewTx) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

//msgp:tuple MessageNewSequencer
type MessageNewSequencer struct {
	RawSequencer *RawSequencer
	//Filter       *BloomFilter
	//Hop          uint8
}

func (m *MessageNewSequencer) GetType() msg.BinaryMessageType {
	return MessageTypeNewSequencer
}

func (m *MessageNewSequencer) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageNewSequencer) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageNewSequencer) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

//msgp:tuple MessageNewTxs
type MessageNewTxs struct {
	RawTxs *RawTxs
}

func (m *MessageNewTxs) GetType() msg.BinaryMessageType {
	return MessageTypeNewTxs
}

func (m *MessageNewTxs) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageNewTxs) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageNewTxs) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
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

func (z *MessageTxsRequest) GetType() msg.BinaryMessageType {
	return MessageTypeTxsRequest
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
	RawSequencer *RawSequencer
	//RawCampaigns   *RawCampaigns
	//RawTermChanges *RawTermChanges
	RawTxs      *TxisMarshaler
	RequestedId uint32 //avoid msg drop
}

func (m *MessageTxsResponse) GetType() msg.BinaryMessageType {
	return MessageTypeTxsResponse
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
	Origin    HashOrNumber // Block from which to retrieve headers
	Amount    uint64       // Maximum number of headers to retrieve
	Skip      uint64       // Blocks to skip between consecutive headers
	Reverse   bool         // Query direction (false = rising towards latest, true = falling towards genesis)
	RequestId uint32       //avoid msg drop
}

func (m *MessageHeaderRequest) GetType() msg.BinaryMessageType {
	return MessageTypeHeaderRequest
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
	return fmt.Sprintf("Origin: [%s],amount : %d ,skip : %d, reverse : %v, requestId :%d", m.Origin.String(), m.Amount, m.Skip, m.Reverse, m.RequestId)

}

//msgp:tuple MessageSequencerHeader
type MessageSequencerHeader struct {
	Hash   *common.Hash
	Number *uint64
}

func (m *MessageSequencerHeader) GetType() msg.BinaryMessageType {
	return MessageTypeSequencerHeader
}

func (m *MessageSequencerHeader) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageSequencerHeader) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageSequencerHeader) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageSequencerHeader) String() string {
	return fmt.Sprintf("hash: %s, number : %d", m.Hash.String(), m.Number)
}

//msgp:tuple MessageHeaderResponse
type MessageHeaderResponse struct {
	Headers     *SequencerHeaders
	RequestedId uint32 //avoid msg drop
}

func (m *MessageHeaderResponse) GetType() msg.BinaryMessageType {
	return MessageTypeHeaderResponse
}

func (m *MessageHeaderResponse) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageHeaderResponse) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageHeaderResponse) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageHeaderResponse) String() string {
	return fmt.Sprintf("headers: [%s] reuqestedId :%d", m.Headers.String(), m.RequestedId)
}

//msgp:tuple MessageBodiesRequest
type MessageBodiesRequest struct {
	SeqHashes common.Hashes
	RequestId uint32 //avoid msg drop
}

func (m *MessageBodiesRequest) GetType() msg.BinaryMessageType {
	return MessageTypeBodiesRequest
}

func (m *MessageBodiesRequest) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageBodiesRequest) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageBodiesRequest) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageBodiesRequest) String() string {
	return m.SeqHashes.String() + fmt.Sprintf(" requestId :%d", m.RequestId)
}

//msgp:tuple MessageBodiesResponse
type MessageBodiesResponse struct {
	Bodies      []RawData
	RequestedId uint32 //avoid msg drop
}

func (m *MessageBodiesResponse) GetType() msg.BinaryMessageType {
	return MessageTypeBodiesResponse
}

func (m *MessageBodiesResponse) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageBodiesResponse) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageBodiesResponse) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageBodiesResponse) String() string {
	return fmt.Sprintf("bodies len : %d, reuqestedId :%d", len(m.Bodies), m.RequestedId)
}

//msgp:tuple MessageControl
type MessageControl struct {
	Hash *common.Hash
}

func (m *MessageControl) GetType() msg.BinaryMessageType {
	return MessageTypeControl
}

func (m *MessageControl) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageControl) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageControl) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageControl) String() string {
	if m == nil || m.Hash == nil {
		return ""
	}
	return m.Hash.String()
}

//msgp:tuple MessageGetMsg
type MessageGetMsg struct {
	Hash *common.Hash
}

func (m *MessageGetMsg) GetType() msg.BinaryMessageType {
	return MessageTypeGetMsg
}

func (m *MessageGetMsg) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageGetMsg) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageGetMsg) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageGetMsg) String() string {
	if m == nil || m.Hash == nil {
		return ""
	}
	return m.Hash.String()
}

//msgp:tuple MessageGetMsg
type MessageDuplicate bool

func (m *MessageDuplicate) GetType() msg.BinaryMessageType {
	return MessageTypeDuplicate
}

func (m *MessageDuplicate) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageDuplicate) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageDuplicate) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageDuplicate) String() string {
	return "MessageDuplicate"
}

//msgp:tuple MessageCampaign
type MessageCampaign struct {
	RawCampaign *RawCampaign
}

func (m *MessageCampaign) GetType() msg.BinaryMessageType {
	return MessageTypeCampaign
}

func (m *MessageCampaign) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageCampaign) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageCampaign) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageCampaign) String() string {
	return m.RawCampaign.String()
}

//msgp:tuple MessageTermChange
type MessageTermChange struct {
	RawTermChange *RawTermChange
}

func (m *MessageTermChange) GetType() msg.BinaryMessageType {
	return MessageTypeTermChange
}

func (m *MessageTermChange) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageTermChange) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageTermChange) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageTermChange) String() string {
	return m.RawTermChange.String()
}

//msgp:tuple MessageTermChangeRequest
type MessageTermChangeRequest struct {
	Id uint32
}

func (m *MessageTermChangeRequest) GetType() msg.BinaryMessageType {
	return MessageTypeTermChangeRequest
}

func (m *MessageTermChangeRequest) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageTermChangeRequest) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageTermChangeRequest) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageTermChangeRequest) String() string {
	return fmt.Sprintf("requst id %d ", m.Id)
}

//msgp:tuple MessageTermChangeResponse
type MessageTermChangeResponse struct {
	TermChange *TermChange
	Id         uint32
}

func (m *MessageTermChangeResponse) GetType() msg.BinaryMessageType {
	return MessageTypeTermChangeResponse
}

func (m *MessageTermChangeResponse) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageTermChangeResponse) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageTermChangeResponse) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageTermChangeResponse) String() string {
	return fmt.Sprintf("requst id %d , %v ", m.Id, m.TermChange)
}

//msgp:tuple MessageNewActionTx
type MessageNewActionTx struct {
	ActionTx *ActionTx
}

func (m *MessageNewActionTx) GetType() msg.BinaryMessageType {
	return MessageTypeNewActionTx
}

func (m *MessageNewActionTx) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageNewActionTx) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageNewActionTx) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageNewActionTx) String() string {
	if m.ActionTx == nil {
		return "nil"
	}
	return m.ActionTx.String()
}

type MessagePlain struct {
	InnerMessageType msg.BinaryMessageType
	InnerMessage     []byte
}

func (m MessagePlain) GetType() msg.BinaryMessageType {
	return MessageTypePlain
}

func (m MessagePlain) GetData() []byte {
	return m.InnerMessage
}

func (m MessagePlain) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.InnerMessageType,
		Data: m.InnerMessage,
	}
}

func (m MessagePlain) FromBinary(bs []byte) error {
	m.InnerMessageType = MessageTypePlain
	m.InnerMessage = bs
	return nil
}

func (m MessagePlain) String() string {
	return "MessagePlain"
}

//msgp:tuple MessageSigned
type MessageSigned struct {
	InnerMessageType msg.BinaryMessageType
	InnerMessage     []byte
	Signature        hexutil.Bytes
	PublicKey        hexutil.Bytes
	TermId           uint32
}

func (m MessageSigned) GetType() msg.BinaryMessageType {
	return MessageTypeSigned
}

func (m MessageSigned) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m MessageSigned) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m MessageSigned) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m MessageSigned) String() string {
	return "MessageSigned"
}

//msgp:tuple MessageEncrypted
type MessageEncrypted struct {
	InnerMessageType      msg.BinaryMessageType
	InnerMessageEncrypted []byte
	PublicKey             hexutil.Bytes
}

func (m *MessageEncrypted) GetType() msg.BinaryMessageType {
	return MessageTypeEncrypted
}

func (m *MessageEncrypted) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageEncrypted) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageEncrypted) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageEncrypted) String() string {
	return "MessageEncrypted"

}
