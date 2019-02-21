package dkg

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	vss "github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/vss/pedersen"
	"testing"
)

func TestDeal_UnmarshalBinary(t *testing.T) {
	v := vss.EncryptedDeal{
		Cipher:randomBytes(15),
		Signature:randomBytes(10),
	}
	d:= Deal{
		Index:895623,
		Deal:&v,
		Signature:randomBytes(4),
	}
	data,_  := d.MarshalBinary()
	fmt.Println(len(data))
	fmt.Println(hex.EncodeToString(data))
	var d2 Deal
	err := d2.UnmarshalBinary(data)
	if err!=nil || d2.Index !=d.Index  {
		t.Fatal(err,d,d2)
	}
	fmt.Println(d,d2)
}

func TestReadWrite(t *testing.T) {
	var a ,c int32
	a = 57
	var b bytes.Buffer
	err:= binary.Write(&b,binary.LittleEndian,a)
 fmt.Println(err)
	d := b.Bytes()
	fmt.Println(hex.EncodeToString(d))

	r:= bytes.NewReader(d)

	err = binary.Read(r,binary.LittleEndian,&c )
	fmt.Println(a,c,err)
}
