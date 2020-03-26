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
