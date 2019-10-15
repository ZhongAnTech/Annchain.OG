package archive

import (
	"github.com/annchain/OG/types/msg"
)

//go:generate msgp

//msgp:tuple MessageNewArchive
type MessageNewArchive struct {
	Archive *Archive
}

func (m *MessageNewArchive) GetType() msg.BinaryMessageType {
	return msg.MessageTypeNewArchive
}

func (m *MessageNewArchive) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageNewArchive) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageNewArchive) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageNewArchive) String() string {
	if m.Archive == nil {
		return "nil"
	}
	return m.Archive.String()
}