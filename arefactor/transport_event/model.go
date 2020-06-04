package transport_event

//go:generate msgp

// WireMessage is for transportation.
//msgp WireMessage
type WireMessage struct {
	MsgType      int    // what ContentByte is.
	ContentBytes []byte // this Byte will be recovered to implementation of Content interface
}

//func (z *WireMessage) String() string {
//	return fmt.Sprintf("WM: Type=%d OriginId=%s ContentBytes=%s", z.MsgType, PrettyId(z.OriginId), ToBriefHex(z.ContentBytes, 100))
//}
