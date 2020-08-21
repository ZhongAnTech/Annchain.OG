package ogsyncer_interface

type ResourceType uint8

//go:generate msgp

const (
	ResourceTypeTx ResourceType = iota
	ResourceTypeSequencer
	ResourceTypeArchive
	ResourceTypeAction
	ResourceTypeInt
)

var ResourceTypeStrings = map[ResourceType]string{
	ResourceTypeTx:        "RTx",
	ResourceTypeSequencer: "RTs",
	ResourceTypeInt:       "RTi",
}

//msgp MessageContentTx
type MessageContentTx struct {
	Hash         []byte
	ParentsHash  [][]byte
	MineNonce    uint64
	AccountNonce uint64
	From         []byte
	To           []byte
	Value        string // bigint
	TokenId      int32
	PublicKey    []byte
	Data         []byte
	Signature    []byte
}

func (z *MessageContentTx) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessageContentTx) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp MessageContentSequencer
type MessageContentSequencer struct {
	Hash         []byte
	ParentsHash  [][]byte
	MineNonce    uint64
	AccountNonce uint64
	Issuer       []byte
	PublicKey    []byte
	Signature    []byte
	StateRoot    []byte
	Height       uint64
}

func (z *MessageContentSequencer) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessageContentSequencer) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

//msgp MessageContentInt
type MessageContentInt struct {
	Height      int64
	Step        int
	PreviousSum int
	MySum       int
	Submitter   int
	Ts          string
}

func (z *MessageContentInt) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessageContentInt) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}