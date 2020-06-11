package og_interface

// sync

type UnknownManager interface {
}

type Syncer interface {
}

const (
	UnknownTypeHeight UnknownType = iota
	UnknownTypeHash
)

type UnknownType int

// Unknown represents a resource that needs to be synced from others
type Unknown interface {
	GetType() UnknownType
	GetValue() interface{}
}

type UnknownHeight struct {
	value int
}

func (u UnknownHeight) GetType() UnknownType {
	return UnknownTypeHeight
}

func (u UnknownHeight) GetValue() interface{} {
	return u.value
}

type UnknownHash struct {
	value Hash
}

func (u UnknownHash) GetType() UnknownType {
	return UnknownTypeHash
}

func (u UnknownHash) GetValue() interface{} {
	return u.value
}
