// general_message is the package for low level p2p communications
// the content in BinaryMessage should either be sent directly to the others,
// or be wrapped by another BinaryMessage.
// all high level messages should have a way to convert itself to the low level format

package msg

//go:generate msgp

type OgMessageType uint8

// OgMessage is the message that can be convert to BinaryMessage
type OgMessage interface {
	GetType() OgMessageType
	String() string

	Marshal() []byte
	Unmarshal([]byte) error
}
