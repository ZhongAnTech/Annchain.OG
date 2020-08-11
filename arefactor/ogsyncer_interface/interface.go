package ogsyncer_interface

import (
	"fmt"
	"github.com/annchain/OG/arefactor/og_interface"
)

type Resource interface {
	GetType() int
	ToBytes() []byte
	FromBytes([]byte) error
}

type ResourceRequest interface {
	GetType() int
	ToBytes() []byte
	FromBytes([]byte) error
}

type ResourceFetcher interface {
	Fetch(request ResourceRequest) []Resource
}

type OgSyncMessage interface {
	GetType() OgSyncMessageType
	GetTypeValue() int
	ToBytes() []byte
	FromBytes(bts []byte) error
	String() string
}

// sync

type UnknownManager interface {
	Enqueue(task Unknown, hint SourceHint)
}

type Syncer interface {
	NeedToKnow(unknown Unknown, hint SourceHint)
}

type SourceHint struct {
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
	GetId() string
}

type UnknownHeight struct {
	value int
}

func (u UnknownHeight) GetId() string {
	return fmt.Sprintf("HT.%d", u.value)
}

func (u UnknownHeight) GetType() UnknownType {
	return UnknownTypeHeight
}

func (u UnknownHeight) GetValue() interface{} {
	return u.value
}

type UnknownHash struct {
	value og_interface.Hash
}

func (u UnknownHash) GetId() string {
	return fmt.Sprintf("HS.%s", u.value)
}

func (u UnknownHash) GetType() UnknownType {
	return UnknownTypeHash
}

func (u UnknownHash) GetValue() interface{} {
	return u.value
}
