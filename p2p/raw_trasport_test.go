package p2p

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	msg2 "github.com/annchain/OG/common/msg"
	"testing"
	"github.com/annchain/OG/common/hexutil"
)

func TestRawFrameRW_ReadMsg(t *testing.T) {
	buf := new(bytes.Buffer)
	rw := newRawFrameRW(buf)
	golden := unhex(`000006c28080089401020304`)

	// Check WriteMsg. This puts a message into the buffer.
	a := msg2.Uints{1, 2, 3, 4}
	b, _ := a.MarshalMsg(nil)
	fmt.Println(hexutil.Encode(b),len(b))
	if err := Send(rw, 8, b); err != nil {
		t.Fatalf("WriteMsg error: %v", err)
	}
	written := buf.Bytes()
	fmt.Println(len(written), hex.EncodeToString(written))
	if !bytes.Equal(written, golden) {
		t.Fatalf("output mismatch:\n  got:  %x\n  want: %x", written, golden)
	}

	// Check ReadMsg. It reads the message encoded by WriteMsg, which
	// is equivalent to the golden message above.
	msg, err := rw.ReadMsg()
	if err != nil {
		t.Fatalf("ReadMsg error: %v", err)
	}
	if msg.Size != 5 {
		t.Errorf("msg size mismatch: got %d, want %d", msg.Size, 5)
	}
	if msg.Code != 8 {
		t.Errorf("msg code mismatch: got %d, want %d", msg.Code, 8)
	}
	payload, _ := ioutil.ReadAll(msg.Payload)
	wantPayload := unhex("9401020304")
	if !bytes.Equal(payload, wantPayload) {
		t.Errorf("msg payload mismatch:\ngot  %x\nwant %x", payload, wantPayload)
	}
}
