// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/annchain/OG/common/hexutil"
	msg2 "github.com/annchain/OG/common/msg"
	"io"
	"io/ioutil"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/ecies"
	"github.com/annchain/OG/common/crypto/sha3"
	"github.com/davecgh/go-spew/spew"
)

func TestSharedSecret(t *testing.T) {
	prv0, _ := crypto.GenerateKey() // = ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	pub0 := &prv0.PublicKey
	prv1, _ := crypto.GenerateKey()
	pub1 := &prv1.PublicKey

	ss0, err := ecies.ImportECDSA(prv0).GenerateShared(ecies.ImportECDSAPublic(pub1), sskLen, sskLen)
	if err != nil {
		return
	}
	ss1, err := ecies.ImportECDSA(prv1).GenerateShared(ecies.ImportECDSAPublic(pub0), sskLen, sskLen)
	if err != nil {
		return
	}
	t.Logf("Secret:\n%v %x\n%v %x", len(ss0), ss0, len(ss0), ss1)
	if !bytes.Equal(ss0, ss1) {
		t.Errorf("dont match :(")
	}
}

func TestEncHandshake(t *testing.T) {
	for i := 0; i < 10; i++ {
		start := time.Now()
		if err := testEncHandshake(nil); err != nil {
			t.Fatalf("i=%d %v", i, err)
		}
		t.Logf("(without token) %d %v\n", i+1, time.Since(start))
	}
	for i := 0; i < 10; i++ {
		tok := make([]byte, shaLen)
		rand.Reader.Read(tok)
		start := time.Now()
		if err := testEncHandshake(tok); err != nil {
			t.Fatalf("i=%d %v", i, err)
		}
		t.Logf("(with token) %d %v\n", i+1, time.Since(start))
	}
}

func testEncHandshake(token []byte) error {
	type result struct {
		side   string
		pubkey *ecdsa.PublicKey
		err    error
	}
	var (
		prv0, _  = crypto.GenerateKey()
		prv1, _  = crypto.GenerateKey()
		fd0, fd1 = net.Pipe()
		c0, c1   = newRLPX(fd0).(*rlpx), newRLPX(fd1).(*rlpx)
		output   = make(chan result)
	)

	go func() {
		r := result{side: "initiator"}
		defer func() { output <- r }()
		defer fd0.Close()

		r.pubkey, r.err = c0.doEncHandshake(prv0, &prv1.PublicKey)
		if r.err != nil {
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

	// compare derived secrets
	if !reflect.DeepEqual(c0.rw.egressMAC, c1.rw.ingressMAC) {
		return fmt.Errorf("egress mac mismatch:\n c0.rw: %#v\n c1.rw: %#v", c0.rw.egressMAC, c1.rw.ingressMAC)
	}
	if !reflect.DeepEqual(c0.rw.ingressMAC, c1.rw.egressMAC) {
		return fmt.Errorf("ingress mac mismatch:\n c0.rw: %#v\n c1.rw: %#v", c0.rw.ingressMAC, c1.rw.egressMAC)
	}
	if !reflect.DeepEqual(c0.rw.enc, c1.rw.enc) {
		return fmt.Errorf("enc cipher mismatch:\n c0.rw: %#v\n c1.rw: %#v", c0.rw.enc, c1.rw.enc)
	}
	if !reflect.DeepEqual(c0.rw.dec, c1.rw.dec) {
		return fmt.Errorf("dec cipher mismatch:\n c0.rw: %#v\n c1.rw: %#v", c0.rw.dec, c1.rw.dec)
	}
	return nil
}

/*
func TestProtocolHandshake(t *testing.T) {
	var (
		prv0, _ = crypto.GenerateKey()
		node0   = &discover.Node{ID: discover.PubkeyID(&prv0.PublicKey), IP: net.IP{1, 2, 3, 4}, TCP: 33}
		hs0     = &protoHandshake{Version: 3, ID: node0.ID, Caps: []Cap{{"a", 0}, {"b", 2}}}

		prv1, _ = crypto.GenerateKey()
		node1   = &discover.Node{ID: discover.PubkeyID(&prv1.PublicKey), IP: net.IP{5, 6, 7, 8}, TCP: 44}
		hs1     = &protoHandshake{Version: 3, ID: node1.ID, Caps: []Cap{{"c", 1}, {"d", 3}}}

		wg sync.WaitGroup
	)

	fd0, fd1, err := pipes.TCPPipe()
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		defer fd0.Close()
		rlpx := newRLPX(fd0)
		remid, err := rlpx.doEncHandshake(prv0, node1)
		if err != nil {
			t.Errorf("dial side enc handshake failed: %v", err)
			return
		}
		if remid != node1.ID {
			t.Errorf("dial side remote id mismatch: got %v, want %v", remid, node1.ID)
			return
		}

		phs, err := rlpx.doProtoHandshake(hs0)
		if err != nil {
			t.Errorf("dial side proto handshake error: %v", err)
			return
		}
		phs.Rest = nil
		if !reflect.DeepEqual(phs, hs1) {
			t.Errorf("dial side proto handshake mismatch:\ngot: %s\nwant: %s\n", spew.Sdump(phs), spew.Sdump(hs1))
			return
		}
		rlpx.close(DiscQuitting)
	}()
	go func() {
		defer wg.Done()
		defer fd1.Close()
		rlpx := newRLPX(fd1)
		remid, err := rlpx.doEncHandshake(prv1, nil)
		if err != nil {
			t.Errorf("listen side enc handshake failed: %v", err)
			return
		}
		if remid != node0.ID {
			t.Errorf("listen side remote id mismatch: got %v, want %v", remid, node0.ID)
			return
		}

		phs, err := rlpx.doProtoHandshake(hs1)
		if err != nil {
			t.Errorf("listen side proto handshake error: %v", err)
			return
		}
		phs.Rest = nil
		if !reflect.DeepEqual(phs, hs0) {
			t.Errorf("listen side proto handshake mismatch:\ngot: %s\nwant: %s\n", spew.Sdump(phs), spew.Sdump(hs0))
			return
		}

		if err := ExpectMsg(rlpx, discMsg, []DiscReason{DiscQuitting}); err != nil {
			t.Errorf("error receiving disconnect: %v", err)
		}
	}()
	wg.Wait()
}
*/
func TestProtocolHandshakeErrors(t *testing.T) {
	our := &ProtoHandshake{Version: 3, Caps: []Cap{{"foo", 2}, {"bar", 3}}, Name: "quux"}
	quitData, _ := DiscQuitting.MarshalMsg(nil)
	pro := ProtoHandshake{Version: 3}
	protoData, _ := pro.MarshalMsg(nil)
	tests := []struct {
		code MsgCodeType
		//msg  interface{}
		msg []byte
		err error
	}{
		{
			code: discMsg,
			//msg:  []DiscReason{DiscQuitting},
			msg: quitData,
			err: DiscQuitting,
		},
		{
			code: 0x9898,
			msg:  []byte{1},
			err:  errors.New("expected handshake, got 989898"),
		},
		{
			code: handshakeMsg,
			msg:  make([]byte, baseProtocolMaxMsgSize+2),
			err:  errors.New("message too big"),
		},
		{
			code: handshakeMsg,
			msg:  []byte{1, 2, 3},
			err:  newPeerError(errInvalidMsg, "(code 0) (size 3) msgp: attempted to decode type \"int\" with method for \"map\""),
		},
		{
			code: handshakeMsg,
			//msg:  &ProtoHandshake{Version: 3},
			msg: protoData,
			err: DiscInvalidIdentity,
		},
	}

	for i, test := range tests {
		p1, p2 := MsgPipe()
		go Send(p1, test.code, test.msg)
		_, err := readProtocolHandshake(p2, our)
		if !reflect.DeepEqual(err, test.err) {
			t.Errorf("test %d: error mismatch: got %q, want %q", i, err, test.err)
		}
	}
}

func TestRlpxFrameRW_ReadMsg(t *testing.T) {
	buf := new(bytes.Buffer)
	hash := fakeHash([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	rw := newRLPXFrameRW(buf, secrets{
		AES:        crypto.Keccak256(),
		MAC:        crypto.Keccak256(),
		IngressMAC: hash,
		EgressMAC:  hash,
	})
	golden:= unhex(`0082bbdae471818bb0bfa6b551d1cb4201010101010101010101010101010101ba
	62a6f3941e80e6674fccb640b769410ca260cb7b00
	9e5b45d021af17dc638f146f174d0e5ff380abf27f3a87a6e4f101010101010101010101010101010101`)
	// Check WriteMsg. This puts a message into the buffer.
	a := msg2.Bytes(unhex("ba328a4ba590cb43f7848f41c438288500828ddae471818bb0bfa6b551d1cb42465456465485658686555555ab"))
	b, _ := a.MarshalMsg(nil)
	fmt.Println(hexutil.Encode(b), len(b))
	if err := Send(rw, 8, b); err != nil {
		t.Fatalf("WriteMsg error: %v", err)
	}
	var code MsgCodeType
	code = 8
	codeM,_  := code.MarshalMsg(nil)
	fmt.Println(codeM, len(codeM))
	written := buf.Bytes()
	fmt.Println(len(written),hex.EncodeToString(written))
	if !bytes.Equal(written, golden) {
		t.Fatalf("output mismatch:\n  got:  %x\n  want: %x", written, golden)
	}
	// Check ReadMsg. It reads the message encoded by WriteMsg, which
	// is equivalent to the golden message above.
	msg, err := rw.ReadMsg()
	if err != nil {
		t.Fatalf("ReadMsg error: %v", err)
	}
	if msg.Size != 47 {
		t.Errorf("msg size mismatch: got %d, want %d", msg.Size, 47)
	}
	fmt.Println("siez ",msg.Size)
	if msg.Code != 8 {
		t.Errorf("msg code mismatch: got %d, want %d", msg.Code, 8)
	}
	payload, _ := ioutil.ReadAll(msg.Payload)
	wantPayload := unhex("c42dba328a4ba590cb43f7848f41c438288500828ddae471818bb0bfa6b551d1cb42465456465485658686555555ab")
	if !bytes.Equal(payload, wantPayload) {
		t.Errorf("msg payload mismatch:\ngot  %x\nwant %x", payload, wantPayload)
	}
}

func TestRLPXFrameFake(t *testing.T) {
	buf := new(bytes.Buffer)
	hash := fakeHash([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	rw := newRLPXFrameRW(buf, secrets{
		AES:        crypto.Keccak256(),
		MAC:        crypto.Keccak256(),
		IngressMAC: hash,
		EgressMAC:  hash,
	})

	golden := unhex(`
00828ddae471818bb0bfa6b551d1cb42
01010101010101010101010101010101
ba328a4ba590cb43f7848f41c4382885
01010101010101010101010101010101
`)

	// Check WriteMsg. This puts a message into the buffer.
	a := msg2.Uints{1, 2, 3, 4}
	b, _ := a.MarshalMsg(nil)
	if err := Send(rw, 8, b); err != nil {
		t.Fatalf("WriteMsg error: %v", err)
	}
	written := buf.Bytes()
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

type fakeHash []byte

func (fakeHash) Write(p []byte) (int, error) { return len(p), nil }
func (fakeHash) Reset()                      {}
func (fakeHash) BlockSize() int              { return 0 }

func (h fakeHash) Size() int           { return len(h) }
func (h fakeHash) Sum(b []byte) []byte { return append(b, h...) }

func TestRLPXFrameRW(t *testing.T) {
	var (
		aesSecret      = make([]byte, 16)
		macSecret      = make([]byte, 16)
		egressMACinit  = make([]byte, 32)
		ingressMACinit = make([]byte, 32)
	)
	for _, s := range [][]byte{aesSecret, macSecret, egressMACinit, ingressMACinit} {
		rand.Read(s)
	}
	conn := new(bytes.Buffer)

	s1 := secrets{
		AES:        aesSecret,
		MAC:        macSecret,
		EgressMAC:  sha3.NewKeccak256(),
		IngressMAC: sha3.NewKeccak256(),
	}
	s1.EgressMAC.Write(egressMACinit)
	s1.IngressMAC.Write(ingressMACinit)
	rw1 := newRLPXFrameRW(conn, s1)

	s2 := secrets{
		AES:        aesSecret,
		MAC:        macSecret,
		EgressMAC:  sha3.NewKeccak256(),
		IngressMAC: sha3.NewKeccak256(),
	}
	s2.EgressMAC.Write(ingressMACinit)
	s2.IngressMAC.Write(egressMACinit)
	rw2 := newRLPXFrameRW(conn, s2)

	// send some messages
	for i := 0; i < 10; i++ {
		// write message into conn buffer
		wmsg := msg2.Strings{"foo", "bar", strings.Repeat("test", i)}
		b, _ := wmsg.MarshalMsg(nil)
		err := Send(rw1, MsgCodeType(i), b)
		if err != nil {
			t.Fatalf("WriteMsg error (i=%d): %v", i, err)
		}

		// read message that rw1 just wrote
		msg, err := rw2.ReadMsg()
		if err != nil {
			t.Fatalf("ReadMsg error (i=%d): %v", i, err)
		}
		if msg.Code != MsgCodeType(i) {
			t.Fatalf("msg code mismatch: got %d, want %d", msg.Code, i)
		}
		payload, _ := ioutil.ReadAll(msg.Payload)
		wantPayload, _ := wmsg.MarshalMsg(nil)
		//wantPayload, _ := rlp.EncodeToBytes(wmsg)
		if !bytes.Equal(payload, wantPayload) {
			t.Fatalf("msg payload mismatch:\ngot  %x\nwant %x", payload, wantPayload)
		}
	}
}

type handshakeAuthTest struct {
	input       string
	isPlain     bool
	wantVersion uint
	wantRest    [][]byte
}

var eip8HandshakeAuthTests = []handshakeAuthTest{
	// (Auth₁) RLPx v4 plain encoding
	{
		input: `
			048ca79ad18e4b0659fab4853fe5bc58eb83992980f4c9cc147d2aa31532efd29a3d3dc6a3d89eaf
			913150cfc777ce0ce4af2758bf4810235f6e6ceccfee1acc6b22c005e9e3a49d6448610a58e98744
			ba3ac0399e82692d67c1f58849050b3024e21a52c9d3b01d871ff5f210817912773e610443a9ef14
			2e91cdba0bd77b5fdf0769b05671fc35f83d83e4d3b0b000c6b2a1b1bba89e0fc51bf4e460df3105
			c444f14be226458940d6061c296350937ffd5e3acaceeaaefd3c6f74be8e23e0f45163cc7ebd7622
			0f0128410fd05250273156d548a414444ae2f7dea4dfca2d43c057adb701a715bf59f6fb66b2d1d2
			0f2c703f851cbf5ac47396d9ca65b6260bd141ac4d53e2de585a73d1750780db4c9ee4cd4d225173
			a4592ee77e2bd94d0be3691f3b406f9bba9b591fc63facc016bfa8
		`,
		isPlain:     true,
		wantVersion: 4,
	},
	// (Auth₂) EIP-8 encoding
	{
		input: `
		 01fd04e21891e689df83f64e0f6fd092c3060aeaebeae4abd52898524a6dde5167
aeecce9bc44234c8953db2b084ee7f8f0773e43861469cc1021b2154aed092988d49ff079abe
52b24069b344b08e0680dd0d388b53e2651f86259c91eaad261e09d5be94c070f91c0ae3cb1af
fa11017aabdc223a5919596da1999260715eda05f3653623c05350fa7bf06dcde5a754066f8d4
64a3fa99c47d50e0ff3b7b58ad43e85ceaca2646cc36206e0b0648b92604b6593559d202d1b294
60af08d6818decf069118ded9eaabac157fa77494b90c23d94a3932694bbdd65abcfbe44000b
1b5e19dec5345968756f7be241b614f239613bfd1b073d23f83bb9916013f02bd815800cdded
09c429ced8526d0b433bf400e54eb30ec5ce9dc6f96a75dee773afbbb73d64ffc9dc5db4080a
5571b3ad9122f67a24815e53d43522209606800d9ff8dd29411ce279d40ec12f202370356396
62248f1ce553bab7b35aa13bd76a0ae435fc3515a5708b4f8f71af5bf96657e04d572850c6cb
241fc2dccdfd4f9fc04ce5d53e6e502f2fc9477aabaa829ee26e2e2f377a434d5e2e6e6632b
8100013c90c05828fa14e6dbc2d32a10eed51877ad5cf06396220bab227405f48dc725c2bc
2281f762212dc192d672cb5bc3289ac50a0244c68d33f509d49d8789744464dc2909fffda0
31155bad633ad7b30eb43263aa565c086f326a8cea4c5
		`,
		wantVersion: 4,
		//wantRest:    [][]byte{},
	},
	// (Auth₃) RLPx v4 EIP-8 encoding with version 56, additional list elements
	{
		input: `
		 020e0439c7b8803f6c8939c83ab2e024093a2ab2e5834512f47bf84aa33439f03539053a84cec7469803d
		4edeeea66059be82d4fa0aa419d3ab77631c228f1936e27e57ce6c5dc9a785fd19892367ecb21a716fbf8d
     498280924d63c2b27977797ff71742fa9960f8e9f6571045c5a1e812644f541221aac59c5e27d5daba1f8119db944f
      1466d6c0e068bbd7621b18ec237be63ed26f0e5b664b3395d344a3c0c57d89653816329e9e2ffd79e01ce4b5614ea
      e80755e0aca3655eb3a435c1e7a443e0af36a9908b3dd076d01905d4c9224e0f941ebdd51e8d4
    7fb4f495ea88c928e176ced61917da8f18bcca3365970df65f357d099dc720fdaf4aa3aaaf779af0074425551c38222c2899353028d92bf
   ecf2b8003cf35d4bcc9ce2a25dcdc79d79cc0022835515938a59ee79bfbd78c6bef57fda03c0c42231783c7445015f247a248cdcd5a61a2
   e108929197d3e0e83bd6d2f025115fb21e890193b3c752880a6e1758e5ded5633d81de771f68659e77511b36bac3892868148a332c3d9fb
   1f6e060a94c10d6466ab43d44b7fde993aba6542a3d16ddbbf1308ff8b4ad04ec3b9856953931e3b18d15036deb415796fb3ec06cfb9341
    c0072a1d60d1f416b57fd7d7800ad6923b4497aeb3210f5af001f731c7c65a22b282eecfdc6da46fa9c482a22ffa9c86073a2caf2f4da1
         550fd79c43a03824a4932f032e8d3828844a02fab8eede2030c8d9fb0ea0a408699
		`,
		wantVersion: 56,
		wantRest:    [][]byte{{0x01}, {0x02}, {0xC2, 0x04, 0x05}},
	},
}

type handshakeAckTest struct {
	input       string
	wantVersion uint
	wantRest    [][]byte
}

var eip8HandshakeRespTests = []handshakeAckTest{
	// (Ack₁) RLPx v4 plain encoding
	{
		input: `
			049f8abcfa9c0dc65b982e98af921bc0ba6e4243169348a236abe9df5f93aa69d99cadddaa387662
			b0ff2c08e9006d5a11a278b1b3331e5aaabf0a32f01281b6f4ede0e09a2d5f585b26513cb794d963
			5a57563921c04a9090b4f14ee42be1a5461049af4ea7a7f49bf4c97a352d39c8d02ee4acc416388c
			1c66cec761d2bc1c72da6ba143477f049c9d2dde846c252c111b904f630ac98e51609b3b1f58168d
			dca6505b7196532e5f85b259a20c45e1979491683fee108e9660edbf38f3add489ae73e3dda2c71b
			d1497113d5c755e942d1
		`,
		wantVersion: 4,
	},
	// (Ack₂) EIP-8 encoding
	{
		input: `
0197047250f566862bb97317b79c342e96f8b7ac4deedb677a687b6914dccdb7f283e0ae2ddff9efc799c8d0804ca6
2fbc0681ae93d5e3d4fdbcf573e75bf0fb0237456335bccfeabb0de09cbab881fc055eb1a86b455df2fc0b75d88978
9bb5c8887eed760079a6d0df3cee6947d3f58b6f7fb4a8999bb21bfbf218fdb06a43cc78d1bc48a513999860e8ea99
ff3c1f5182fe89a4df4f3c5b3c578bdc4ec64334f09f78ee3158e6162da9515bb6dd30a9b9d48e78c5f2a6c076e212
41b2a39ea04c439a3d9567ee27e76dc3b89484bf6305536a0485a5e4859c78370b315b57f9f72fd78e880bdbff1b13
b4f750e0343056bed0190cab1da47af608d9fc6cc4f41f631b4a312e5541b89b57e5472407ba5543b1e624b266a597
99a08595b0dbae70bde98718abe5bd361cb396f5e9fc0b04f0157cd9bf002075a27a4b7e86a6c8a6895831d4d366b3
1204ef1112b53c1aac130a2a1477a952a3b346154f02cd609e90545a4e12a0df4f8a1cf39296ab2b319aa7d61952d6d9
2f8173f5d8081e8c69e05ca50ec9e831bb3ce5301100ee9e975f0ef34db54a30
		`,
		wantVersion: 4,
		//wantRest:    [][]byte{},
	},
	// (Ack₃) EIP-8 encoding with version 57, additional list elements
	{
		input: `
			01b90454cd2aa35b7e6b4498bff8297dde8394fe1184cc72b4d09bfba53b5f23da6b7a265db5b6ff40e954f7
f0fd4c4a5466ed7a1ac15d6d7904f8aafa9c3d965a84e405e7da79dd1150bd0b17036697a6dfcd6a36a5819b45a706a3955f
bb247d9fb1d7a3fa2d93088d7d4f34f99a5dc889033251b953b41c4d3c89633947b0bdf836ea908d4e290db26b76b7d2a403
25c3e54373d7a654cb6839cb42fccf815763db6800268a5609643cb43ff32f7e20ccfe2a448c726cf8196e4327a071abaee5
255a6d9b42807e87a322d4dccd155ef9cd1f02ca38c5dc3a8cf12272247449e85760db4042ce08f94729a02c36bae454e99c
ba0ec489dd9ebc747b41ae38854605bd40e92c758403dec8dbc3b76422970015f3a63b7fa5d077cfd353dc06a0d01263ad59
da6da1acd564f9d4a062c5569e111950307f90cf594980cc0d5ee7daa1a85218c94759243a7a0510e32dc2ad4a23c4c201f7
bab86a05baa126ce572b01d345f21c6011b9e2aebad983b3beaaa66c2a4c729012ca691b9fe38cbe2b97c0a7120b85fdcd27
2343b4116b98a71d535b471bb61a704f553ba4bfb70dbd77f6226262ed90135b842c23ae152e84aac09feb991ba04aacbe
		`,
		wantVersion: 57,
		wantRest:    [][]byte{{0x06}, {0xC2, 0x07, 0x08}, {0x81, 0xFA}},
	},
}

func TestHandshakeForwardCompatibility(t *testing.T) {
	var (
		keyA, _       = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		keyB, _       = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		pubA          = crypto.FromECDSAPub(&keyA.PublicKey)[1:]
		pubB          = crypto.FromECDSAPub(&keyB.PublicKey)[1:]
		ephA, _       = crypto.HexToECDSA("869d6ecf5211f1cc60418a13b9d870b22959d0c16f02bec714c960dd2298a32d")
		ephB, _       = crypto.HexToECDSA("e238eb8e04fee6511ab04c6dd3c89ce097b11f25d584863ac2b6d5b35b1847e4")
		ephPubA       = crypto.FromECDSAPub(&ephA.PublicKey)[1:]
		ephPubB       = crypto.FromECDSAPub(&ephB.PublicKey)[1:]
		nonceA        = unhex("7e968bba13b6c50e2c4cd7f241cc0d64d1ac25c7f5952df231ac6a2bda8ee5d6")
		nonceB        = unhex("559aead08264d5795d3909718cdd05abd49572e84fe55590eef31a88a08fdffd")
		_, _, _, _    = pubA, pubB, ephPubA, ephPubB
		authSignature = unhex("299ca6acfd35e3d72d8ba3d1e2b60b5561d5af5218eb5bc182045769eb4226910a301acae3b369fffc4a4899d6b02531e89fd4fe36a2cf0d93607ba470b50f7800")
		_             = authSignature
	)
	makeAuth := func(test handshakeAuthTest) *AuthMsgV4 {
		msg := &AuthMsgV4{Version: test.wantVersion, Rest: test.wantRest, gotPlain: test.isPlain}
		copy(msg.Signature[:], authSignature)
		copy(msg.InitiatorPubkey[:], pubA)
		copy(msg.Nonce[:], nonceA)
		return msg
	}
	makeAck := func(test handshakeAckTest) *AuthRespV4 {
		msg := &AuthRespV4{Version: test.wantVersion, Rest: test.wantRest}
		copy(msg.RandomPubkey[:], ephPubB)
		copy(msg.Nonce[:], nonceB)
		return msg
	}

	// check auth msg parsing
	for _, test := range eip8HandshakeAuthTests {
		r := bytes.NewReader(unhex(test.input))
		msg := new(AuthMsgV4)
		/*
			var someText []byte
			msg2 := makeAuth(test)
			pub_b:=  ecies.ImportECDSAPublic(&keyB.PublicKey)
			if !msg2.gotPlain{
				someText,_= preEip8test(msg2,pub_b)
				r2 := bytes.NewReader(someText)
				fmt.Println(r2.Size())
			}

		*/
		ciphertext, err := readHandshakeMsg(msg, encAuthMsgLen, keyB, r)
		if err != nil {
			t.Errorf("error for input %x:\n  %v ", unhex(test.input), err)
			continue
		}
		if !bytes.Equal(ciphertext, unhex(test.input)) {
			t.Errorf("wrong ciphertext for input %x:\n  %x", unhex(test.input), ciphertext)
		}
		want := makeAuth(test)
		if !reflect.DeepEqual(msg, want) {
			t.Errorf("wrong msg for input %x:\ngot %s\nwant %s", unhex(test.input), spew.Sdump(msg), spew.Sdump(want))
			fmt.Println(msg.Version != want.Version, msg.gotPlain)
		}
	}

	// check auth resp parsing
	for _, test := range eip8HandshakeRespTests {
		input := unhex(test.input)
		r := bytes.NewReader(input)
		msg := new(AuthRespV4)
		/*
			var someText []byte
			msg2 := makeAck(test)
			pub_a:=  ecies.ImportECDSAPublic(&keyA.PublicKey)

				someText,_= preEip8Resptest(msg2,pub_a)
				r = bytes.NewReader(someText)
				fmt.Println(r.Size())

		*/
		ciphertext, err := readHandshakeMsgResp(msg, encAuthRespLen, keyA, r)
		if err != nil {
			t.Errorf("error for input %x:\n  %v", input, err)
			continue
		}
		if !bytes.Equal(ciphertext, input) {
			t.Errorf("wrong ciphertext for input %x:\n  %x", input, ciphertext)
		}
		want := makeAck(test)
		if !reflect.DeepEqual(msg, want) {
			t.Errorf("wrong msg for input %x:\ngot %s\nwant %s", input, spew.Sdump(msg), spew.Sdump(want))
		}
	}

	// check derivation for (Auth₂, Ack₂) on recipient side
	var (
		hs = &encHandshake{
			initiator:     false,
			respNonce:     nonceB,
			randomPrivKey: ecies.ImportECDSA(ephB),
		}
		authCiphertext     = unhex(eip8HandshakeAuthTests[1].input)
		authRespCiphertext = unhex(eip8HandshakeRespTests[1].input)
		authMsg            = makeAuth(eip8HandshakeAuthTests[1])
		wantAES            = unhex("80e8632c05fed6fc2a13b0f8d31a3cf645366239170ea067065aba8e28bac487")
		wantMAC            = unhex("2ea74ec5dae199227dff1af715362700e989d889d7a493cb0639691efb8e5f98")
		//wantFooIngressHash = unhex("0c7ec6340062cc46f5e9f1e3cf86f8c8c403c5a0964f5df0ebd34a75ddc86db5")
		wantFooIngressHash = unhex("d402a923042e8aab1f01f0783c343d574d3a0d70872e874168ea4d6174129e77")
	)
	if err := hs.handleAuthMsg(authMsg, keyB); err != nil {
		t.Fatalf("handleAuthMsg: %v", err)
	}
	derived, err := hs.secrets(authCiphertext, authRespCiphertext)
	if err != nil {
		t.Fatalf("secrets: %v", err)
	}
	if !bytes.Equal(derived.AES, wantAES) {
		t.Errorf("aes-secret mismatch:\ngot %x\nwant %x", derived.AES, wantAES)
	}
	if !bytes.Equal(derived.MAC, wantMAC) {
		t.Errorf("mac-secret mismatch:\ngot %x\nwant %x", derived.MAC, wantMAC)
	}
	io.WriteString(derived.IngressMAC, "foo")
	fooIngressHash := derived.IngressMAC.Sum(nil)
	if !bytes.Equal(fooIngressHash, wantFooIngressHash) {
		t.Errorf("ingress-mac('foo') mismatch:\ngot %x\nwant %x", fooIngressHash, wantFooIngressHash)
	}
}
