package annsensus

import (
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/types/msg"
)

//go:generate msgp

type AnnsensusMessageType uint16

// all message type that is supported by annsensus should be listed here
const (
	AnnsensusMessageTypePlain AnnsensusMessageType = iota + 100
	AnnsensusMessageTypeSigned
	AnnsensusMessageTypeEncrypted
)

//msgp:tuple AnnsensusMessagePlain
type AnnsensusMessagePlain struct {
	InnerMessageType AnnsensusMessageType
	InnerMessage     []byte
}

func (z *AnnsensusMessagePlain) GetType() AnnsensusMessageType {
	return AnnsensusMessageTypePlain
}

func (z *AnnsensusMessagePlain) String() string {
	return "AnnsensusMessagePlain"
}

//msgp:tuple AnnsensusMessageSigned
type AnnsensusMessageSigned struct {
	InnerMessageType AnnsensusMessageType
	InnerMessage     []byte
	Signature        hexutil.Bytes
	PublicKey        hexutil.Bytes
	TermId           uint32
}

func (m *AnnsensusMessageSigned) GetType() AnnsensusMessageType {
	return AnnsensusMessageTypeSigned
}

func (m *AnnsensusMessageSigned) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *AnnsensusMessageSigned) String() string {
	return "AnnsensusMessageSigned"
}

//msgp:tuple AnnsensusMessageEncrypted
type AnnsensusMessageEncrypted struct {
	InnerMessageType      msg.BinaryMessageType
	InnerMessageEncrypted []byte
	PublicKey             hexutil.Bytes
}

func (m *AnnsensusMessageEncrypted) GetType() AnnsensusMessageType {
	return AnnsensusMessageTypeEncrypted
}

func (m *AnnsensusMessageEncrypted) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

//
//func (m *AnnsensusMessageEncrypted) ToBinary() msg.BinaryMessage {
//	return msg.BinaryMessage{
//		Type: m.GetType(),
//		Data: m.GetData(),
//	}
//}
//
//func (m *AnnsensusMessageEncrypted) FromBinary(bs []byte) error {
//	_, err := m.UnmarshalMsg(bs)
//	return err
//}

func (m *AnnsensusMessageEncrypted) String() string {
	return "AnnsensusMessageEncrypted " + hexutil.Encode(m.InnerMessageEncrypted)

}
