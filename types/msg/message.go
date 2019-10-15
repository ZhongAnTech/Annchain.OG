// general_message is the package for low level p2p communications
// the content in BinaryMessage should either be sent directly to the others,
// or be wrapped by another BinaryMessage.
// all high level messages should have a way to convert itself to the low level format

package msg

type BinaryMessageType uint16

// BinaryMessage stores data that can be directly sent to the others, or be wrapped by another BinaryMessage.
type BinaryMessage struct {
	Type BinaryMessageType
	Data []byte
}

// TransportableMessage is the message that can be convert to BinaryMessage
type TransportableMessage interface {
	GetType() BinaryMessageType
	GetData() []byte
	ToBinary() BinaryMessage
	FromBinary([]byte) error
	String() string
}
