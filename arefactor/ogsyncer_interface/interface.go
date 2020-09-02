package ogsyncer_interface

import (
	"fmt"
	"github.com/annchain/OG/arefactor/og_interface"
)

//type Resource interface {
//	GetType() int
//	ToBytes() []byte
//	FromBytes([]byte) error
//}
//
//type ResourceRequest interface {
//	GetType() int
//	ToBytes() []byte
//	FromBytes([]byte) error
//}
//
//type ResourceFetcher interface {
//	Fetch(request ResourceRequest) []Resource
//}

type SequencerReceivedEventArg struct {
}

type TxReceivedEventArg struct {
}

type IntsReceivedEventArg struct {
	Ints MessageContentInt
	From string
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
	GetHintPeerId() string
}

type UnknownHeight struct {
	Height     int64
	HintPeerId string
}

func (u UnknownHeight) GetHintPeerId() string {
	return u.HintPeerId
}

func (u UnknownHeight) GetId() string {
	return fmt.Sprintf("HT.%d", u.Height)
}

func (u UnknownHeight) GetType() UnknownType {
	return UnknownTypeHeight
}

func (u UnknownHeight) GetValue() interface{} {
	return u.Height
}

type UnknownHash struct {
	Hash       og_interface.Hash
	HintPeerId string
}

func (u UnknownHash) GetHintPeerId() string {
	return u.HintPeerId
}

func (u UnknownHash) GetId() string {
	return fmt.Sprintf("HS.%s", u.Hash)
}

func (u UnknownHash) GetType() UnknownType {
	return UnknownTypeHash
}

func (u UnknownHash) GetValue() interface{} {
	return u.Hash
}

type UnknownNeededEventSubscriber interface {
	Name() string
	UnknownNeededEventChannel() chan Unknown
}
