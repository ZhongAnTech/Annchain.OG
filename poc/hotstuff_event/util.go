package hotstuff_event

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/tinylib/msgp/msgp"
)

type Hashable interface {
	GetHashContent() string
}

// Msg is NOT for transportation. It is only an internal structure
type Msg struct {
	Typev    MsgType
	Sig      Signature
	SenderId string // no need to fill it when sending
	Content  Content
	//ParentQC *QC
	//Round    int
	//Id       int //

	//ViewNumber    int
	//Node          *Node
	//Justify       *QC
	//FromPartnerId int
}

func (m Msg) String() string {
	return fmt.Sprintf("[type:%s sender=%d content=%s sig=%s]", m.Typev, m.SenderId, m.Content, m.Sig)
}

type Content interface {
	fmt.Stringer
	msgp.Marshaler
	msgp.Unmarshaler
	msgp.Decodable
	msgp.Encodable
	msgp.Sizer
	SignatureTarget() string
}

func Hash(s string) string {
	bs := md5.Sum([]byte(s))
	return hex.EncodeToString(bs[:])
}

type NilInt struct {
	Value int
}

func ToBriefHex(bytes []byte, maxLen int) string {
	if maxLen >= len(bytes) {
		return hex.EncodeToString(bytes)
	}
	return hex.EncodeToString(bytes[0:maxLen/2]) + "..." + hex.EncodeToString(bytes[maxLen/2:len(bytes)])
}
