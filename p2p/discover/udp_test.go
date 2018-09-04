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

package discover

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/davecgh/go-spew/spew"

	"github.com/annchain/OG/common"
	"github.com/tinylib/msgp/msgp"
)

func init() {
	spew.Config.DisableMethods = true
}

// shared test variables
var (
	futureExp          = uint64(time.Now().Add(10 * time.Hour).Unix())
	testTarget         = NodeID{0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1}
	testRemote         = RpcEndpoint{IP: net.ParseIP("1.1.1.1").To4(), UDP: 1, TCP: 2}
	testLocalAnnounced = RpcEndpoint{IP: net.ParseIP("2.2.2.2").To4(), UDP: 3, TCP: 4}
	testLocal          = RpcEndpoint{IP: net.ParseIP("3.3.3.3").To4(), UDP: 5, TCP: 6}
)

type udpTest struct {
	t                   *testing.T
	pipe                *dgramPipe
	table               *Table
	udp                 *udp
	sent                [][]byte
	localkey, remotekey *ecdsa.PrivateKey
	remoteaddr          *net.UDPAddr
}

func newUDPTest(t *testing.T) *udpTest {
	test := &udpTest{
		t:          t,
		pipe:       newpipe(),
		localkey:   newkey(),
		remotekey:  newkey(),
		remoteaddr: &net.UDPAddr{IP: net.IP{10, 0, 1, 99}, Port: 30303},
	}
	test.table, test.udp, _ = newUDP(test.pipe, Config{PrivateKey: test.localkey})
	// Wait for initial refresh so the table doesn't send unexpected findnode.
	<-test.table.initDone
	return test
}

// handles a packet as if it had been sent to the transport.
func (test *udpTest) packetIn(wantError error, ptype byte, data []byte) error {
	enc, _, err := encodePacket(test.remotekey, ptype, data)
	if err != nil {
		return test.errorf("packet (%d) encode error: %v", ptype, err)
	}
	test.sent = append(test.sent, enc)
	if err = test.udp.handlePacket(test.remoteaddr, enc); err != wantError {
		return test.errorf("error mismatch: got %q, want %q", err, wantError)
	}
	return nil
}

// waits for a packet to be sent by the transport.
// validate should have type func(*udpTest, X) error, where X is a packet type.
func (test *udpTest) waitPacketOut(validate interface{}) ([]byte, error) {
	dgram := test.pipe.waitPacketOut()
	p, _, hash, err := decodePacket(dgram)
	if err != nil {
		return hash, test.errorf("sent packet decode error: %v", err)
	}
	fn := reflect.ValueOf(validate)
	exptype := fn.Type().In(0)
	if reflect.TypeOf(p) != exptype {
		return hash, test.errorf("sent packet type mismatch, got: %v, want: %v", reflect.TypeOf(p), exptype)
	}
	fn.Call([]reflect.Value{reflect.ValueOf(p)})
	return hash, nil
}

func (test *udpTest) errorf(format string, args ...interface{}) error {
	_, file, line, ok := runtime.Caller(2) // errorf + waitPacketOut
	if ok {
		file = filepath.Base(file)
	} else {
		file = "???"
		line = 1
	}
	err := fmt.Errorf(format, args...)
	fmt.Printf("\t%s:%d: %v\n", file, line, err)
	test.t.Fail()
	return err
}

func TestUDP_packetErrors(t *testing.T) {
	test := newUDPTest(t)
	defer test.table.Close()
	ping := &Ping{From: testRemote, To: testLocalAnnounced, Version: 4}
	data, _ := ping.MarshalMsg(nil)
	test.packetIn(errExpired, pingPacket, data)
	pong := &Pong{ReplyTok: []byte{}, Expiration: futureExp}
	data, _ = pong.MarshalMsg(nil)
	test.packetIn(errUnsolicitedReply, pongPacket, data)
	f := &Findnode{Expiration: futureExp}
	data, _ = f.MarshalMsg(nil)
	test.packetIn(errUnknownNode, findnodePacket, data)
	n := &Neighbors{Expiration: futureExp}
	data, _ = n.MarshalMsg(nil)
	test.packetIn(errUnsolicitedReply, neighborsPacket, data)
}

func TestUDP_pingTimeout(t *testing.T) {
	t.Parallel()
	test := newUDPTest(t)
	defer test.table.Close()

	toaddr := &net.UDPAddr{IP: net.ParseIP("1.2.3.4"), Port: 2222}
	toid := NodeID{1, 2, 3, 4}
	if err := test.udp.ping(toid, toaddr); err != errTimeout {
		t.Error("expected timeout error, got", err)
	}
}

func TestUDP_responseTimeouts(t *testing.T) {
	t.Parallel()
	test := newUDPTest(t)
	defer test.table.Close()

	rand.Seed(time.Now().UnixNano())
	randomDuration := func(max time.Duration) time.Duration {
		return time.Duration(rand.Int63n(int64(max)))
	}

	var (
		nReqs      = 200
		nTimeouts  = 0                       // number of requests with ptype > 128
		nilErr     = make(chan error, nReqs) // for requests that get a reply
		timeoutErr = make(chan error, nReqs) // for requests that time out
	)
	for i := 0; i < nReqs; i++ {
		// Create a matcher for a random request in udp.loop. Requests
		// with ptype <= 128 will not get a reply and should time out.
		// For all other requests, a reply is scheduled to arrive
		// within the timeout window.
		p := &pending{
			ptype:    byte(rand.Intn(255)),
			callback: func(interface{}) bool { return true },
		}
		binary.BigEndian.PutUint64(p.from[:], uint64(i))
		if p.ptype <= 128 {
			p.errc = timeoutErr
			test.udp.addpending <- p
			nTimeouts++
		} else {
			p.errc = nilErr
			test.udp.addpending <- p
			time.AfterFunc(randomDuration(60*time.Millisecond), func() {
				if !test.udp.handleReply(p.from, p.ptype, nil) {
					t.Logf("not matched: %v", p)
				}
			})
		}
		time.Sleep(randomDuration(30 * time.Millisecond))
	}

	// Check that all timeouts were delivered and that the rest got nil errors.
	// The replies must be delivered.
	var (
		recvDeadline        = time.After(20 * time.Second)
		nTimeoutsRecv, nNil = 0, 0
	)
	for i := 0; i < nReqs; i++ {
		select {
		case err := <-timeoutErr:
			if err != errTimeout {
				t.Fatalf("got non-timeout error on timeoutErr %d: %v", i, err)
			}
			nTimeoutsRecv++
		case err := <-nilErr:
			if err != nil {
				t.Fatalf("got non-nil error on nilErr %d: %v", i, err)
			}
			nNil++
		case <-recvDeadline:
			t.Fatalf("exceeded recv deadline")
		}
	}
	if nTimeoutsRecv != nTimeouts {
		t.Errorf("wrong number of timeout errors received: got %d, want %d", nTimeoutsRecv, nTimeouts)
	}
	if nNil != nReqs-nTimeouts {
		t.Errorf("wrong number of successful replies: got %d, want %d", nNil, nReqs-nTimeouts)
	}
}

func TestUDP_findnodeTimeout(t *testing.T) {
	t.Parallel()
	test := newUDPTest(t)
	defer test.table.Close()

	toaddr := &net.UDPAddr{IP: net.ParseIP("1.2.3.4"), Port: 2222}
	toid := NodeID{1, 2, 3, 4}
	target := NodeID{4, 5, 6, 7}
	result, err := test.udp.findnode(toid, toaddr, target)
	if err != errTimeout {
		t.Error("expected timeout error, got", err)
	}
	if len(result) > 0 {
		t.Error("expected empty result, got", result)
	}
}

func TestUDP_findnode(t *testing.T) {
	test := newUDPTest(t)
	defer test.table.Close()
	//logrus.SetLevel(logrus.DebugLevel)
	// put a few nodes into the table. their exact
	// distribution shouldn't matter much, although we need to
	// take care not to overflow any bucket.
	targetHash := crypto.Keccak256Hash(testTarget[:])
	nodes := &nodesByDistance{target: targetHash}
	for i := 0; i < bucketSize; i++ {
		nodes.push(nodeAtDistance(test.table.self.sha, i+2), bucketSize)
	}
	test.table.stuff(nodes.entries)

	// ensure there's a bond with the test node,
	// findnode won't be accepted otherwise.
	test.table.db.updateLastPongReceived(PubkeyID(&test.remotekey.PublicKey), time.Now())

	// check that closest Neighbors are returned.
	f := &Findnode{Target: testTarget, Expiration: futureExp}
	data, _ := f.MarshalMsg(nil)
	err := test.packetIn(nil, findnodePacket, data)
	if err != nil {
		t.Errorf(" packed in error,%v", err)
	}
	expected := test.table.closest(targetHash, bucketSize)
	waitNeighbors := func(want []*Node) {
		test.waitPacketOut(func(p *Neighbors) {
			fmt.Println(len(p.Nodes), len(want), bucketSize)
			if len(p.Nodes) != len(want) {
				t.Errorf("wrong number of results: got %d, want %d", len(p.Nodes), bucketSize)
			}
			for i := range p.Nodes {
				if p.Nodes[i].ID != want[i].ID {
					t.Errorf("result mismatch at %d:\n  got:  %v\n  want: %v", i, p.Nodes[i], expected.entries[i])
				}
			}
		})
	}
	waitNeighbors(expected.entries[:maxNeighbors])
	waitNeighbors(expected.entries[maxNeighbors:])
}

func TestUDP_findnodeMultiReply(t *testing.T) {
	test := newUDPTest(t)
	defer test.table.Close()

	rid := PubkeyID(&test.remotekey.PublicKey)
	test.table.db.updateLastPingReceived(rid, time.Now())

	// queue a pending findnode request
	resultc, errc := make(chan []*Node), make(chan error)
	go func() {
		ns, err := test.udp.findnode(rid, test.remoteaddr, testTarget)
		if err != nil && len(ns) == 0 {
			errc <- err
		} else {
			resultc <- ns
		}
	}()

	// wait for the findnode to be sent.
	// after it is sent, the transport is waiting for a reply
	test.waitPacketOut(func(p *Findnode) {
		if p.Target != testTarget {
			t.Errorf("wrong target: got %v, want %v", p.Target, testTarget)
		}
	})

	// send the reply as two packets.
	list := []*Node{
		MustParseNode("enode://ba85011c70bcc5c04d8607d3a0ed29aa6179c092cbdda10d5d32684fb33ed01bd94f588ca8f91ac48318087dcb02eaf36773a7a453f0eedd6742af668097b29c@10.0.1.16:30303?discport=30304"),
		MustParseNode("enode://81fa361d25f157cd421c60dcc28d8dac5ef6a89476633339c5df30287474520caca09627da18543d9079b5b288698b542d56167aa5c09111e55acdbbdf2ef799@10.0.1.16:30303"),
		MustParseNode("enode://9bffefd833d53fac8e652415f4973bee289e8b1a5c6c4cbe70abf817ce8a64cee11b823b66a987f51aaa9fba0d6a91b3e6bf0d5a5d1042de8e9eeea057b217f8@10.0.1.36:30301?discport=17"),
		MustParseNode("enode://1b5b4aa662d7cb44a7221bfba67302590b643028197a7d5214790f3bac7aaa4a3241be9e83c09cf1f6c69d007c634faae3dc1b1221793e8446c0b3a09de65960@10.0.1.16:30303"),
	}
	rpclist := make([]RpcNode, len(list))
	for i := range list {
		rpclist[i] = nodeToRPC(list[i])
	}
	n := &Neighbors{Expiration: futureExp, Nodes: rpclist[:2]}

	data, _ := n.MarshalMsg(nil)
	test.packetIn(nil, neighborsPacket, data)
	n = &Neighbors{Expiration: futureExp, Nodes: rpclist[2:]}
	data, _ = n.MarshalMsg(nil)
	test.packetIn(nil, neighborsPacket, data)

	// check that the sent Neighbors are all returned by FindNode
	select {
	case result := <-resultc:
		want := append(list[:2], list[3:]...)
		if !reflect.DeepEqual(result, want) {
			t.Errorf("neighbors mismatch:\n  got:  %v\n  want: %v", result, want)
		}
	case err := <-errc:
		t.Errorf("FindNode error: %v", err)
	case <-time.After(5 * time.Second):
		t.Error("FindNode did not return within 5 seconds")
	}
}

func TestUDP_successfulPing(t *testing.T) {
	test := newUDPTest(t)
	added := make(chan *Node, 1)
	test.table.nodeAddedHook = func(n *Node) { added <- n }
	defer test.table.Close()

	// The remote side sends a ping packet to initiate the exchange.
	ping := &Ping{From: testRemote, To: testLocalAnnounced, Version: 4, Expiration: futureExp}
	data, _ := ping.MarshalMsg(nil)
	go test.packetIn(nil, pingPacket, data)

	// the Ping is replied to.
	test.waitPacketOut(func(p *Pong) {
		pinghash := test.sent[0][:macSize]
		if !bytes.Equal(p.ReplyTok, pinghash) {
			t.Errorf("got pong.ReplyTok %x, want %x", p.ReplyTok, pinghash)
		}
		wantTo := RpcEndpoint{
			// The mirrored UDP address is the UDP packet sender
			IP: test.remoteaddr.IP, UDP: uint16(test.remoteaddr.Port),
			// The mirrored TCP port is the one from the ping packet
			TCP: testRemote.TCP,
		}
		if !reflect.DeepEqual(p.To, wantTo) {
			t.Errorf("got pong.To %v, want %v", p.To, wantTo)
		}
	})

	// remote is unknown, the table pings back.
	hash, _ := test.waitPacketOut(func(p *Ping) error {
		if !reflect.DeepEqual(p.From, test.udp.ourEndpoint) {
			t.Errorf("got ping.From %v, want %v", p.From, test.udp.ourEndpoint)
		}
		wantTo := RpcEndpoint{
			// The mirrored UDP address is the UDP packet sender.
			IP: test.remoteaddr.IP, UDP: uint16(test.remoteaddr.Port),
			TCP: 0,
		}
		if !reflect.DeepEqual(p.To, wantTo) {
			t.Errorf("got ping.To %v, want %v", p.To, wantTo)
		}
		return nil
	})
	pong := &Pong{ReplyTok: hash, Expiration: futureExp}
	data, _ = pong.MarshalMsg(nil)
	test.packetIn(nil, pongPacket, data)

	// the node should be added to the table shortly after getting the
	// pong packet.
	select {
	case n := <-added:
		rid := PubkeyID(&test.remotekey.PublicKey)
		if n.ID != rid {
			t.Errorf("node has wrong ID: got %v, want %v", n.ID, rid)
		}
		IP := net.IP(n.IP)
		if !IP.Equal(test.remoteaddr.IP) {
			t.Errorf("node has wrong IP: got %v, want: %v", n.IP, test.remoteaddr.IP)
		}
		if int(n.UDP) != test.remoteaddr.Port {
			t.Errorf("node has wrong UDP port: got %v, want: %v", n.UDP, test.remoteaddr.Port)
		}
		if n.TCP != testRemote.TCP {
			t.Errorf("node has wrong TCP port: got %v, want: %v", n.TCP, testRemote.TCP)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("node was not added within 2 seconds")
	}
}

var testPackets = []struct {
	input      string
	wantPacket msgp.Marshaler
	packType   int
}{
	{
		input: "21e70d63e55471f7e8410bb841269d3a67425fc0c76ac3cc82500eefa78815d05baeec3fc55af7af47d552bfb9db5034d89c6a8db837e91675b1c40688b725cf5c505634e749818d78fe10b79716328546f78b6a423224dcb8df4df91142ba87000185a756657273696f6e04a446726f6d83a24950c4047f000001a3554450cd0cfaa3544350cd15a8a2546f83a24950c41000000000000000000000000000000001a3554450cd08aea3544350cd0d05aa45787069726174696f6ece43b9a355a45265737490",
		wantPacket: &Ping{
			Version:    4,
			From:       RpcEndpoint{net.ParseIP("127.0.0.1").To4(), 3322, 5544},
			To:         RpcEndpoint{net.ParseIP("::1"), 2222, 3333},
			Expiration: 1136239445,
			//Rest:       [][]byte{},
		},
		packType: pingPacket,
	},
	{
		input: "ff5f626e33b8fdeaec559ab2f2de7bbea012bdd9ce758a4fca5bc6e75c9133c9196e5af1926532f26f76f96939537624ec85542ae51f7a96f7965d43f4e52b61237858910318584737925ce190c7115a5f406354d571d3a751cc6159c020082d000185a756657273696f6e04a446726f6d83a24950c4047f000001a3554450cd0cfaa3544350cd15a8a2546f83a24950c41000000000000000000000000000000001a3554450cd08aea3544350cd0d05aa45787069726174696f6ece43b9a355a45265737492c40101c40102",
		wantPacket: &Ping{
			Version:    4,
			From:       RpcEndpoint{net.ParseIP("127.0.0.1").To4(), 3322, 5544},
			To:         RpcEndpoint{net.ParseIP("::1"), 2222, 3333},
			Expiration: 1136239445,
			Rest:       [][]byte{{0x01}, {0x02}},
		},
		packType: pingPacket,
	},
	{
		input: "8ae1e77d76a84c4a7209cef8b63bce1a2956e047a7bb188acb47fb5cb482df517c24161961150b933366c951ac83207b7533dfcc7ca7b81a985b76ee7108793b67b2cec384ab6929f05648714b4d982d8fad7908ad8d59c260908af35ca64f7f010185a756657273696f6ecd022ba446726f6d83a24950c41020010db83c4d001500000000abcdef12a3554450cd0cfaa3544350cd15a8a2546f83a24950c41020010db885a308d313198a2e03707348a3554450cd08aea3544350cd823aaa45787069726174696f6ece43b9a355a45265737491c406c50102030405",
		wantPacket: &Ping{
			Version:    555,
			From:       RpcEndpoint{net.ParseIP("2001:db8:3c4d:15::abcd:ef12"), 3322, 5544},
			To:         RpcEndpoint{net.ParseIP("2001:db8:85a3:8d3:1319:8a2e:370:7348"), 2222, 33338},
			Expiration: 1136239445,
			Rest:       [][]byte{{0xC5, 0x01, 0x02, 0x03, 0x04, 0x05}},
		},
		packType: pingPacket,
	},
	{
		input: "ed5adfba32fa90880278ed38b5a3ef707dd983620d9920c1cd4e25c5c31dddf1cac197487422fd912df1ecafc56f20814694ac07cdeab453c3c0dae3295299cf0795c848b48304d01c1f549e5e4985e75501ede042ec8b2bdb3dfb19cb19ff37010284a2546f83a24950c41020010db885a308d313198a2e03707348a3554450cd08aea3544350cd823aa85265706c79546f6bc420fbc914b16819237dcd8801d7e53f69e9719adecb3cc0e790c57e91ca4461c954aa45787069726174696f6ece43b9a355a45265737492c407c6010203c20405c40106",
		wantPacket: &Pong{
			To:         RpcEndpoint{net.ParseIP("2001:db8:85a3:8d3:1319:8a2e:370:7348"), 2222, 33338},
			ReplyTok:   common.Hex2Bytes("fbc914b16819237dcd8801d7e53f69e9719adecb3cc0e790c57e91ca4461c954"),
			Expiration: 1136239445,
			Rest:       [][]byte{{0xC6, 0x01, 0x02, 0x03, 0xC2, 0x04, 0x05}, {0x06}},
		},
		packType: pongPacket,
	},
	{
		input: "ea96bc97610f3173d43533d10776080df59ffa5083318163965d5e67283cda3d2c0a2f361663a4e53b974f16a9fcc42d1fcb85610cec7a335c37d512150793a56c1da83c4761e2ff3086c02dbf5c448cfd7cfab32ff90a8b41d99fc1cc098a1e000383a6546172676574c440ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd31387574077f301b421bc84df7266c44e9e6d569fc56be00812904767bf5ccd1fc7faa45787069726174696f6ece43b9a355a45265737492c403829999c40483999999",
		wantPacket: &Findnode{
			Target:     MustHexID("ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd31387574077f301b421bc84df7266c44e9e6d569fc56be00812904767bf5ccd1fc7f"),
			Expiration: 1136239445,
			Rest:       [][]byte{{0x82, 0x99, 0x99}, {0x83, 0x99, 0x99, 0x99}},
		},
		packType: findnodePacket,
	},
	{
		input: "71ca07eb48dac0d2bcd3dcafa37827e56d1215ca98bccb5eb19d00c954c0451bd8bf9d2126ecc17b83e5290ba5fc9ddd98f04dd4a4a555e2cfd0c9c308dbe4d41fa1026f0e3033a4897650b09dbb24fca5c7cb453bcb6c45855a683e0ac0ec1d000483a54e6f6465739484a24950c40463211637a3554450cd115ca3544350cd115da24944c4403155e1427f85f10a5c9a7755877748041af1bcd8d474ec065eb33df57a97babf54bfd2103575fa829115d224c523596b401065a97f74010610fce76382c0bf3284a24950c40401020304a355445001a354435001a24944c440312c55512422cf9b8a4097e9a6ad79402e87a15ae909a4bfefa22398f03d20951933beea1e4dfa6f968212385e829f04c2d314fc2d4e255e0d3bc08792b069db84a24950c41020010db83c4d001500000000abcdef12a3554450cd0d05a3544350cd0d05a24944c44038643200b172dcfef857492156971f0e6aa2c538d8b74010f8e140811d53b98c765dd2d96126051913f44582e8c199ad7c6d6819e9a56483f637feaac9448aac84a24950c41020010db885a308d313198a2e03707348a3554450cd03e7a3544350cd03e8a24944c4408dcab8618c3253b558d459da53bd8fa68935a719aff8b811197101a4b2b47dd2d47295286fc00cc081bb542d760717d1bdd6bec2c37cd72eca367d6dd3b9df73aa45787069726174696f6ece43b9a355a45265737493c40101c40102c40103",
		wantPacket: &Neighbors{
			Nodes: []RpcNode{
				{
					ID:  MustHexID("3155e1427f85f10a5c9a7755877748041af1bcd8d474ec065eb33df57a97babf54bfd2103575fa829115d224c523596b401065a97f74010610fce76382c0bf32"),
					IP:  net.ParseIP("99.33.22.55").To4(),
					UDP: 4444,
					TCP: 4445,
				},
				{
					ID:  MustHexID("312c55512422cf9b8a4097e9a6ad79402e87a15ae909a4bfefa22398f03d20951933beea1e4dfa6f968212385e829f04c2d314fc2d4e255e0d3bc08792b069db"),
					IP:  net.ParseIP("1.2.3.4").To4(),
					UDP: 1,
					TCP: 1,
				},
				{
					ID:  MustHexID("38643200b172dcfef857492156971f0e6aa2c538d8b74010f8e140811d53b98c765dd2d96126051913f44582e8c199ad7c6d6819e9a56483f637feaac9448aac"),
					IP:  net.ParseIP("2001:db8:3c4d:15::abcd:ef12"),
					UDP: 3333,
					TCP: 3333,
				},
				{
					ID:  MustHexID("8dcab8618c3253b558d459da53bd8fa68935a719aff8b811197101a4b2b47dd2d47295286fc00cc081bb542d760717d1bdd6bec2c37cd72eca367d6dd3b9df73"),
					IP:  net.ParseIP("2001:db8:85a3:8d3:1319:8a2e:370:7348"),
					UDP: 999,
					TCP: 1000,
				},
			},
			Expiration: 1136239445,
			Rest:       [][]byte{{0x01}, {0x02}, {0x03}},
		},
		packType: neighborsPacket,
	},
}

func TestForwardCompatibility(t *testing.T) {
	testkey, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	wantNodeID := PubkeyID(&testkey.PublicKey)

	for _, test := range testPackets {
		input, err := hex.DecodeString(test.input)
		if err != nil {
			t.Fatalf("invalid hex: %s", test.input)
		}

		/*

				data,_:= test.wantPacket.MarshalMsg(nil)
			   pack,_,_ := 	encodePacket(testkey, byte(test.packType),data)
			   fmt.Printf(" data \n %s , input \n %x\n", hex.EncodeToString(pack),input)
			   fmt.Println(bytes.Equal(pack,input))
			   //input = pack
		*/
		packet, nodeid, _, err := decodePacket(input)
		if err != nil {
			t.Errorf("did not accept packet %s\n%v", test.input, err)
			continue
		}
		if !reflect.DeepEqual(packet, test.wantPacket) {
			t.Errorf("got %s\nwant %s", spew.Sdump(packet), spew.Sdump(test.wantPacket))
		}
		if nodeid != wantNodeID {
			t.Errorf("got id %v\nwant id %v", nodeid, wantNodeID)
		}
	}
}

// dgramPipe is a fake UDP socket. It queues all sent datagrams.
type dgramPipe struct {
	mu      *sync.Mutex
	cond    *sync.Cond
	closing chan struct{}
	closed  bool
	queue   [][]byte
}

func newpipe() *dgramPipe {
	mu := new(sync.Mutex)
	return &dgramPipe{
		closing: make(chan struct{}),
		cond:    &sync.Cond{L: mu},
		mu:      mu,
	}
}

// WriteToUDP queues a datagram.
func (c *dgramPipe) WriteToUDP(b []byte, to *net.UDPAddr) (n int, err error) {
	msg := make([]byte, len(b))
	copy(msg, b)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, errors.New("closed")
	}
	c.queue = append(c.queue, msg)
	c.cond.Signal()
	return len(b), nil
}

// ReadFromUDP just hangs until the pipe is closed.
func (c *dgramPipe) ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error) {
	<-c.closing
	return 0, nil, io.EOF
}

func (c *dgramPipe) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		close(c.closing)
		c.closed = true
	}
	return nil
}

func (c *dgramPipe) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: testLocal.IP, Port: int(testLocal.UDP)}
}

func (c *dgramPipe) waitPacketOut() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	for len(c.queue) == 0 {
		c.cond.Wait()
	}
	p := c.queue[0]
	copy(c.queue, c.queue[1:])
	c.queue = c.queue[:len(c.queue)-1]
	return p
}
