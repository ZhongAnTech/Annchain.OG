package archive

import "github.com/annchain/OG/types/general_message"

//go:generate msgp

//msgp:tuple MessageNewArchive
type MessageNewArchive struct {
	Archive *Archive
}

func (m *MessageNewArchive) GetType() general_message.BinaryMessageType {
	return general_message.MessageTypeNewArchive
}

func (m *MessageNewArchive) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageNewArchive) ToBinary() general_message.BinaryMessage {
	return general_message.BinaryMessage{
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