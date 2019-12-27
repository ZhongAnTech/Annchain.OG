package message

type Bytable interface {
	ToBytes() []byte
	FromBytes(bts []byte) error
}
