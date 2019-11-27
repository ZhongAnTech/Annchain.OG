package annsensus

import (
	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp

type AnnsensusMessageType uint16

// all message type that is supported by annsensus should be listed here
const (
	AnnsensusMessageTypeBftPlain AnnsensusMessageType = iota + 100
	AnnsensusMessageTypeBftSigned
	AnnsensusMessageTypeBftEncrypted
	AnnsensusMessageTypeDkgPlain
	AnnsensusMessageTypeDkgSigned
	AnnsensusMessageTypeDkgEncrypted
)

//msgp:tuple AnnsensusMessageBftPlain
type AnnsensusMessageBftPlain struct {
	InnerMessageType uint16 // either bft or dkg type, use uint16 to generalize it
	InnerMessage     []byte
}

func (z *AnnsensusMessageBftPlain) GetData() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *AnnsensusMessageBftPlain) GetType() AnnsensusMessageType {
	return AnnsensusMessageTypeBftPlain
}

func (z *AnnsensusMessageBftPlain) String() string {
	return "AnnsensusMessageTypeBftPlain"
}

//msgp:tuple AnnsensusMessageBftSigned
type AnnsensusMessageBftSigned struct {
	InnerMessageType uint16 // either bft or dkg type, use uint16 to generalize it
	InnerMessage     []byte
	Signature        hexutil.Bytes
	PublicKey        hexutil.Bytes
	TermId           uint32
}

func (m *AnnsensusMessageBftSigned) GetType() AnnsensusMessageType {
	return AnnsensusMessageTypeBftSigned
}

func (m *AnnsensusMessageBftSigned) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *AnnsensusMessageBftSigned) String() string {
	return "AnnsensusMessageBftSigned"
}

//msgp:tuple AnnsensusMessageEncrypted
type AnnsensusMessageBftEncrypted struct {
	InnerMessageType      uint16 // either bft or dkg type, use uint16 to generalize it
	InnerMessageEncrypted []byte
	PublicKey             hexutil.Bytes
}

func (m *AnnsensusMessageBftEncrypted) GetType() AnnsensusMessageType {
	return AnnsensusMessageTypeBftEncrypted
}

func (m *AnnsensusMessageBftEncrypted) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *AnnsensusMessageBftEncrypted) String() string {
	return "AnnsensusMessageBftEncrypted " + hexutil.Encode(m.InnerMessageEncrypted)
}

//msgp:tuple AnnsensusMessageDkgPlain
type AnnsensusMessageDkgPlain struct {
	InnerMessageType uint16 // either bft or dkg type, use uint16 to generalize it
	InnerMessage     []byte
}

func (z *AnnsensusMessageDkgPlain) GetData() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *AnnsensusMessageDkgPlain) GetType() AnnsensusMessageType {
	return AnnsensusMessageTypeDkgPlain
}

func (z *AnnsensusMessageDkgPlain) String() string {
	return "AnnsensusMessageTypeDkgPlain"
}

//msgp:tuple AnnsensusMessageDkgSigned
type AnnsensusMessageDkgSigned struct {
	InnerMessageType uint16 // either bft or dkg type, use uint16 to generalize it
	InnerMessage     []byte
	Signature        hexutil.Bytes
	PublicKey        hexutil.Bytes
	TermId           uint32
}

func (m *AnnsensusMessageDkgSigned) GetType() AnnsensusMessageType {
	return AnnsensusMessageTypeDkgSigned
}

func (m *AnnsensusMessageDkgSigned) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *AnnsensusMessageDkgSigned) String() string {
	return "AnnsensusMessageDkgSigned"
}

//msgp:tuple AnnsensusMessageEncrypted
type AnnsensusMessageDkgEncrypted struct {
	InnerMessageType      uint16 // either bft or dkg type, use uint16 to generalize it
	InnerMessageEncrypted []byte
	PublicKey             hexutil.Bytes
}

func (m *AnnsensusMessageDkgEncrypted) GetType() AnnsensusMessageType {
	return AnnsensusMessageTypeDkgEncrypted
}

func (m *AnnsensusMessageDkgEncrypted) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *AnnsensusMessageDkgEncrypted) String() string {
	return "AnnsensusMessageDkgEncrypted " + hexutil.Encode(m.InnerMessageEncrypted)
}
