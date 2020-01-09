// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	msg2 "github.com/annchain/OG/types/msg"
	"io/ioutil"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestRawFrameRW_ReadMsg(t *testing.T) {
	buf := new(bytes.Buffer)
	rw := newRawFrameRW(buf)
	golden := unhex(`000006c28080089401020304`)

	// Check WriteMsg. This puts a message into the buffer.
	a := msg2.Uints{1, 2, 3, 4}
	b, _ := a.MarshalMsg(nil)
	fmt.Println(hexutil.Encode(b), len(b))
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

func TestRawEncHandshake(t *testing.T) {
	log.Info("hi")
	for i := 0; i < 10; i++ {
		start := time.Now()
		if err := testRawEncHandshake(nil); err != nil {
			t.Fatalf("i=%d %v", i, err)
		}
		t.Logf("(without token) %d %v\n", i+1, time.Since(start))
	}
	for i := 0; i < 10; i++ {
		tok := make([]byte, shaLen)
		rand.Reader.Read(tok)
		start := time.Now()
		if err := testRawEncHandshake(tok); err != nil {
			t.Fatalf("i=%d %v", i, err)
		}
		t.Logf("(with token) %d %v\n", i+1, time.Since(start))
	}
}

func testRawEncHandshake(token []byte) error {
	type result struct {
		side   string
		pubkey *ecdsa.PublicKey
		err    error
	}
	var (
		prv0, _  = crypto.GenerateKey()
		prv1, _  = crypto.GenerateKey()
		fd0, fd1 = net.Pipe()
		c0, c1   = newrawTransport(fd0).(*rawTransport), newrawTransport(fd1).(*rawTransport)
		output   = make(chan result)
	)

	go func() {
		r := result{side: "initiator"}
		defer func() { output <- r }()
		defer fd0.Close()

		r.pubkey, r.err = c0.doEncHandshake(prv0, &prv1.PublicKey)
		if r.err != nil {
			log.WithError(r.err).Error("error")
			return
		}
		if !reflect.DeepEqual(r.pubkey, &prv1.PublicKey) {
			r.err = fmt.Errorf("remote pubkey mismatch: got %v, want: %v", r.pubkey, &prv1.PublicKey)
		}
	}()
	go func() {
		r := result{side: "receiver"}
		defer func() { output <- r }()
		defer fd1.Close()

		r.pubkey, r.err = c1.doEncHandshake(prv1, nil)
		if r.err != nil {
			log.WithError(r.err).Error("error")
			return
		}
		if !reflect.DeepEqual(r.pubkey, &prv0.PublicKey) {
			r.err = fmt.Errorf("remote ID mismatch: got %v, want: %v", r.pubkey, &prv0.PublicKey)
		}
	}()

	// wait for results from both sides
	r1, r2 := <-output, <-output
	if r1.err != nil {
		return fmt.Errorf("%s side error: %v", r1.side, r1.err)
	}
	if r2.err != nil {
		return fmt.Errorf("%s side error: %v", r2.side, r2.err)
	}
	return nil

}

func TestCryptoSignature(t *testing.T) {
	prv, _ := crypto.GenerateKey()
	initNonce := make([]byte, shaLen)
	_, err := rand.Read(initNonce)
	if err != nil {
		t.Fatal(err)
	}
	msg := &RawHandshakeMsg{
		Version: 1,
	}
	copy(msg.Nonce[:], initNonce)
	signature, err := crypto.Sign(initNonce, prv)
	if err != nil {
		t.Fatal(err)
	}
	copy(msg.Signature[:], signature)
	copy(msg.Nonce[:], initNonce)
	//ok := crypto.VerifySignature(msg.InitiatorPubkey[:],msg.Nonce[:],msg.Signature[:])
	res, err := crypto.Ecrecover(msg.Nonce[:], msg.Signature[:])
	log.Debug(hex.EncodeToString(res))
	if err != nil {
		t.Fatal("sig failed ", err, hex.EncodeToString(res))
	}
}
