package bouncer

//go:generate msgp

//msgp BouncerMessage
type BouncerMessage struct {
	Value int
}

func (z *BouncerMessage) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *BouncerMessage) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *BouncerMessage) GetTypeValue() int {
	return 1
}

func (z *BouncerMessage) String() string {
	return "BouncerMessage"
}
