package transport_event

//go:generate msgp

// WireMessage is for transportation.
//msgp WireMessage
type WireMessage struct {
	MsgType      int // what ContentByte is.
	SenderId     string
	ContentBytes []byte // this Byte will be recovered to implementation of Content interface
}

//func (z *WireMessage) String() string {
//	return fmt.Sprintf("WM: Type=%d SenderId=%s ContentBytes=%s", z.MsgType, PrettyId(z.SenderId), ToBriefHex(z.ContentBytes, 100))
//}
