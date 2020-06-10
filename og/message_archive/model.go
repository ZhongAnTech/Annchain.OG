package message_archive

//import (
//	"bytes"
//	"ogcrypto/sha256"
//	"encoding/binary"
//	"errors"
//	"fmt"
//	"github.com/annchain/OG/common"
//	"github.com/annchain/OG/common/ogcrypto"
//	"github.com/sirupsen/logrus"
//)

//type types struct {
//	MessageType    BinaryMessageType
//	Data           []byte
//	Hash           *common.Hash //inner use to avoid resend a Message to the same peer
//	SourceID       string       // the source that this Message  coming from , outgoing if it is nil
//	SendingType    SendingType  //sending type
//	Version        int          // peer Version.
//	Message        p2p_message.Message
//	SourceHash     *common.Hash
//	MarshalState   bool
//	DisableEncrypt bool
//}

//func (m *types) CalculateHash() {
//	// for txs,or response msg , even if  source peer id is different ,they were duplicated txs
//	//for request ,if source id is different they were different  msg ,don't drop it
//	//if we dropped header response because of duplicate , header request will time out
//	if len(m.Data) == 0 {
//		logrus.Error("nil Data to calculate Hash")
//	}
//	data := m.Data
//	var hash *common.Hash
//	switch m.MessageType {
//	case MessageTypeNewTx:
//		msg := m.Message.(*p2p_message.MessageNewTx)
//		hash = msg.GetHash()
//		var msgHash common.Hash
//		msgHash = *hash
//		m.Hash = &msgHash
//		return
//	case MessageTypeControl:
//		msg := m.Message.(*p2p_message.MessageControl)
//		var msgHash common.Hash
//		msgHash = *msg.Hash
//		m.Hash = &msgHash
//		return
//
//	case MessageTypeNewSequencer:
//		msg := m.Message.(*p2p_message.MessageNewSequencer)
//		hash = msg.GetHash()
//		var msgHash common.Hash
//		msgHash = *hash
//		m.Hash = &msgHash
//		return
//	case MessageTypeTxsRequest:
//		data = append(data, []byte(m.SourceID+"txs")...)
//	case MessageTypeBodiesRequest:
//		data = append(data, []byte(m.SourceID+"bq")...)
//	case MessageTypeTermChangeRequest:
//		data = append(data, []byte(m.SourceID+"tq")...)
//	case MessageTypeFetchByHashRequest:
//		data = append(data, []byte(m.SourceID+"fe")...)
//	case MessageTypeHeaderRequest:
//		data = append(data, []byte(m.SourceID+"hq")...)
//	case MessageTypeHeaderResponse:
//		data = append(data, []byte(m.SourceID+"hp")...)
//	case MessageTypeBodiesResponse:
//		data = append(data, []byte(m.SourceID+"bp")...)
//	case MessageTypeSequencerHeader:
//		data = append(data, []byte(m.SourceID+"sq")...)
//	case MessageTypeGetMsg:
//		data = append(data, []byte(m.SourceID+"gm")...)
//	default:
//	}
//	h := sha256.New()
//	h.Write(data)
//	sum := h.Sum(nil)
//	m.Hash = &common.Hash{}
//	m.Hash.MustSetBytes(sum, common.PaddingNone)
//}
//
//func (p *types) GetMarkHashes() common.Hashes {
//	if p.Message == nil {
//		panic("unmarshal first")
//	}
//	switch p.MessageType {
//	case MessageTypeFetchByHashResponse:
//		msg := p.Message.(*p2p_message.MessageSyncResponse)
//		return msg.Hashes()
//	case MessageTypeNewTxs:
//		msg := p.Message.(*p2p_message.MessageNewTxs)
//		return msg.Hashes()
//	case MessageTypeTxsResponse:
//		msg := p.Message.(*p2p_message.MessageTxsResponse)
//		return msg.Hashes()
//	default:
//		return nil
//	}
//	return nil
//}

//func (m *types) Marshal() error {
//	if m.MarshalState {
//		return nil
//	}
//	if m.Message == nil {
//		return errors.New("Message is nil")
//	}
//	var err error
//	m.Data, err = m.Message.MarshalMsg(nil)
//	if err != nil {
//		return err
//	}
//	m.MarshalState = true
//	return err
//}

//func (m *types) AppendGossipTarget(pub *ogcrypto.PublicKey) error {
//	b := make([]byte, 2)
//	//use one key for tx and sequencer
//	binary.BigEndian.PutUint16(b, uint16(m.MessageType))
//	m.Data = append(m.Data, b[:]...)
//	m.DisableEncrypt = true
//	m.Data = append(m.Data, pub.KeyBytes[:8]...)
//	m.MessageType = MessageTypeSecret
//	return nil
//}

//func (m *types) Encrypt(pub *ogcrypto.PublicKey) error {
//	//if m.MessageType == MessageTypeConsensusDkgDeal || m.MessageType == MessageTypeConsensusDkgDealResponse {
//	b := make([]byte, 2)
//	//use one key for tx and sequencer
//	binary.BigEndian.PutUint16(b, uint16(m.MessageType))
//	m.Data = append(m.Data, b[:]...)
//	m.MessageType = MessageTypeSecret
//	ct, err := pub.Encrypt(m.Data)
//	if err != nil {
//		return err
//	}
//	m.Data = ct
//	//add target
//	m.Data = append(m.Data, pub.KeyBytes[:3]...)
//	return nil
//}

//func (m *types) CheckRequiredSize() bool {
//	if m.MessageType == MessageTypeSecret {
//		if m.DisableEncrypt {
//			if len(m.Data) < 8 {
//				return false
//			}
//		}
//		if len(m.Data) < 3 {
//			return false
//		}
//	}
//	return true
//}

//func (m *types) MaybeIsforMe(myPub *ogcrypto.PublicKey) bool {
//	if m.MessageType != MessageTypeSecret {
//		panic("not a secret Message")
//	}
//	//check target
//	if m.DisableEncrypt {
//		target := m.Data[len(m.Data)-8:]
//		if !bytes.Equal(target, myPub.KeyBytes[:8]) {
//			//not four me
//			return false
//		}
//		return true
//	}
//	target := m.Data[len(m.Data)-3:]
//	if !bytes.Equal(target, myPub.KeyBytes[:3]) {
//		//not four me
//		return false
//	}
//	return true
//}

//func (m *types) RemoveGossipTarget() error {
//	msg := make([]byte, len(m.Data)-8)
//	copy(msg, m.Data[:len(m.Data)-8])
//	if len(msg) < 3 {
//		return fmt.Errorf("lengh error %d", len(msg))
//	}
//	b := make([]byte, 2)
//	copy(b, msg[len(msg)-2:])
//	mType := binary.BigEndian.Uint16(b)
//	m.MessageType = BinaryMessageType(mType)
//	if !m.MessageType.IsValid() {
//		return fmt.Errorf("Message type error %s", m.MessageType.String())
//	}
//	m.Data = msg[:len(msg)-2]
//	return nil
//}

//func (m *types) Decrypt(priv *ogcrypto.PrivateKey) error {
//	if m.MessageType != MessageTypeSecret {
//		panic("not a secret Message")
//	}
//	d := make([]byte, len(m.Data)-3)
//	copy(d, m.Data[:len(m.Data)-3])
//	msg, err := priv.Decrypt(d)
//	if err != nil {
//		return err
//	}
//	if len(msg) < 3 {
//		return fmt.Errorf("lengh error %d", len(msg))
//	}
//	b := make([]byte, 2)
//	copy(b, msg[len(msg)-2:])
//	mType := binary.BigEndian.Uint16(b)
//	m.MessageType = BinaryMessageType(mType)
//	if !m.MessageType.IsValid() {
//		return fmt.Errorf("Message type error %s", m.MessageType.String())
//	}
//	m.Data = msg[:len(msg)-2]
//	return nil
//}

//func (p *types) Unmarshal() error {
//	if p.MarshalState {
//		return nil
//	}
//	p.Message = p.MessageType.GetMsg()
//	if p.Message == nil {
//		return fmt.Errorf("unknown Message type %v ", p.MessageType)
//	}
//	switch p.MessageType {
//
//	case MessageTypeNewTx:
//		msg := &p2p_message.MessageNewTx{}
//		_, err := msg.UnmarshalMsg(p.Data)
//		if err != nil {
//			return err
//		}
//		if msg.RawTx == nil {
//			return errors.New("nil content")
//		}
//		p.Message = msg
//		p.MarshalState = true
//		return nil
//	case MessageTypeNewSequencer:
//		msg := &p2p_message.MessageNewSequencer{}
//		_, err := msg.UnmarshalMsg(p.Data)
//		if err != nil {
//			return err
//		}
//		if msg.RawSequencer == nil {
//			return errors.New("nil content")
//		}
//		p.Message = msg
//		p.MarshalState = true
//		return nil
//	case MessageTypeGetMsg:
//		msg := &p2p_message.MessageGetMsg{}
//		_, err := msg.UnmarshalMsg(p.Data)
//		if err != nil {
//			return err
//		}
//		if msg.Hash == nil {
//			return errors.New("nil content")
//		}
//		p.Message = msg
//		p.MarshalState = true
//		return nil
//	case MessageTypeControl:
//		msg := &p2p_message.MessageControl{}
//		_, err := msg.UnmarshalMsg(p.Data)
//		if err != nil {
//			return err
//		}
//		if msg.Hash == nil {
//			return errors.New("nil content")
//		}
//		p.Message = msg
//		p.MarshalState = true
//		return nil
//	default:
//	}
//	_, err := p.Message.UnmarshalMsg(p.Data)
//	p.MarshalState = true
//	return err
//}
//
////
//func (m *types) AllowSendDuplicateMsg() bool {
//	return m.MessageType == MessageTypeNewTx || m.MessageType == MessageTypeNewSequencer
//}
//
//func (m *types) MsgKey() MsgKey {
//	return NewMsgKey(m.MessageType, *m.Hash)
//}


