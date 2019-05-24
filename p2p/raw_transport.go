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
	"errors"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/ecies"
	"github.com/annchain/OG/common/goroutine"
	"github.com/golang/snappy"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"time"
)

//go:generate msgp
type rawTransport struct {
	fd net.Conn
	MsgReadWriter
	rmu, wmu sync.Mutex
	rw       *rawFrameRW
}

func newrawTransport(fd net.Conn) transport {
	fd.SetDeadline(time.Now().Add(handshakeTimeout))
	return &rawTransport{fd: fd}
}

func (t *rawTransport) ReadMsg() (Msg, error) {
	t.rmu.Lock()
	defer t.rmu.Unlock()
	t.fd.SetReadDeadline(time.Now().Add(frameReadTimeout))
	return t.rw.ReadMsg()
}

func (t *rawTransport) WriteMsg(msg Msg) error {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	t.fd.SetWriteDeadline(time.Now().Add(frameWriteTimeout))
	return t.rw.WriteMsg(msg)
}

func (t *rawTransport) close(err error) {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	// Tell the remote end why we're disconnecting if possible.
	if t.rw != nil {
		if r, ok := err.(DiscReason); ok && r != DiscNetworkError {
			// rawTransport tries to send DiscReason to disconnected peer
			// if the connection is net.Pipe (in-memory simulation)
			// it hangs forever, since net.Pipe does not implement
			// a write deadline. Because of this only try to send
			// the disconnect reason message if there is no error.
			if err := t.fd.SetWriteDeadline(time.Now().Add(discWriteTimeout)); err == nil {
				b, _ := r.MarshalMsg(nil)
				Send(t.rw, discMsg, b)
			}
		}
	}
	t.fd.Close()
}

func (t *rawTransport) doProtoHandshake(our *ProtoHandshake) (their *ProtoHandshake, err error) {
	// Writing our handshake happens concurrently, we prefer
	// returning the handshake read error. If the remote side
	// disconnects us early with a valid reason, we should return it
	// as the error so it can be tracked elsewhere.
	werr := make(chan error, 1)
	b, _ := our.MarshalMsg(nil)
	goroutine.New(func() { werr <- Send(t.rw, handshakeMsg, b) })
	if their, err = readProtocolHandshake(t.rw, our); err != nil {
		<-werr // make sure the write terminates too
		return nil, err
	}
	if err := <-werr; err != nil {
		return nil, fmt.Errorf("write error: %v", err)
	}
	// If the protocol version supports Snappy encoding, upgrade immediately
	t.rw.snappy = their.Version >= snappyProtocolVersion

	return their, nil
}

type RawHandshakeMsg struct {
	Signature [sigLen]byte
	Nonce     [shaLen]byte
	Version   uint
}

// RLPx v4 handshake response (defined in EIP-8).
type RawHandshakeResponseMsg struct {
	//RemotePubkey [pubLen]byte
	Nonce [shaLen]byte
	//Signature    [sigLen]byte
	Version uint
}

// rawHandshake contains the state of the encryption handshake.
type rawHandshake struct {
	initiator            bool
	remote               *ecies.PublicKey // remote-pubk
	initNonce, respNonce []byte           // nonce
}

func initiatorRawencHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, remote *ecdsa.PublicKey) (err error) {
	h := &rawHandshake{initiator: true, remote: ecies.ImportECDSAPublic(remote)}
	h.initNonce = make([]byte, shaLen)
	_, err = rand.Read(h.initNonce)
	if err != nil {
		return err
	}
	msg := &RawHandshakeMsg{
		Version: 1,
	}
	copy(msg.Nonce[:], h.initNonce)
	signature, err := crypto.Sign(h.initNonce, prv)
	if err != nil {
		return err
	}
	copy(msg.Signature[:], signature)
	copy(msg.Nonce[:], h.initNonce)
	buf := new(bytes.Buffer)
	b, err := msg.MarshalMsg(nil)
	if err != nil {
		log.WithError(err).Debug("marshal failed")
		return err
	}
	buf.Write(b)
	enc, err := ecies.Encrypt(rand.Reader, h.remote, buf.Bytes(), nil, nil)
	if err != nil {
		log.WithError(err).Debug("enc failed")
		return err
	}
	head := make([]byte, 3)
	putInt24(uint32(len(enc)), head)
	if _, err = conn.Write(head); err != nil {
		log.WithError(err).Debug("write failed")
		return err
	}
	//log.WithField("write len ", len(enc)).WithField("buf size ", len(b)).WithField("msg size ", msg.Msgsize()).Debug("write")
	if _, err = conn.Write(enc); err != nil {
		log.WithError(err).Debug("write failed")
		return err
	}
	log.WithField("nonce ", hex.EncodeToString(msg.Nonce[:])).WithField(
		"sig ", hex.EncodeToString(msg.Signature[:])).Trace("write msg")
	authRespMsg := new(RawHandshakeResponseMsg)
	err = readRawHandshakeMsgResp(authRespMsg, prv, conn)
	if err != nil {
		log.WithError(err).Debug("read response failed")
		return err
	}
	h.respNonce = authRespMsg.Nonce[:]
	return nil
}

func readRawHandshakeMsgResp(msg *RawHandshakeResponseMsg, prv *ecdsa.PrivateKey, r io.Reader) error {
	head := make([]byte, 3)
	if _, err := io.ReadFull(r, head); err != nil {
		log.WithError(err).Debug("read failed")
		return err
	}
	size := readInt24(head)
	if size == 0 {
		return fmt.Errorf("size error")
	}

	buf := make([]byte, size)
	if n, err := io.ReadFull(r, buf); err != nil {
		log.WithError(err).WithField("n ", n).WithField("size", size).Debug("read failed")
		return err
	}
	key := ecies.ImportECDSA(prv)
	dec, err := key.Decrypt(buf, nil, nil)
	if err != nil {
		log.WithError(err).Debug("dec failed")
		return err
	}
	_, err = msg.UnmarshalMsg(dec)
	//	fmt.Println("is plain",msg)
	return nil
}

func readRawHandshakeMsg(msg *RawHandshakeMsg, prv *ecdsa.PrivateKey, r io.Reader) error {
	head := make([]byte, 3)
	if _, err := io.ReadFull(r, head); err != nil {
		log.WithError(err).Debug("read failed")
		return err
	}
	size := readInt24(head)
	if size == 0 {
		return fmt.Errorf("size error")
	}
	buf := make([]byte, size)
	if n, err := io.ReadFull(r, buf); err != nil {
		log.WithError(err).WithField("n ", n).WithField("size", size).Debug("read failed")
		return err
	}

	key := ecies.ImportECDSA(prv)
	dec, err := key.Decrypt(buf, nil, nil)
	if err != nil {
		log.WithField("len ", len(buf)).WithField("hex ", hex.EncodeToString(buf)).WithError(err).Debug("dec failed")
		return err
	}
	_, err = msg.UnmarshalMsg(dec)
	//	fmt.Println("is plain",msg)
	return nil
}

func receiverRawHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey) (*ecdsa.PublicKey, error) {
	authMsg := new(RawHandshakeMsg)
	err := readRawHandshakeMsg(authMsg, prv, conn)
	if err != nil {
		return nil, err
	}
	//log.WithField("nonce ", hex.EncodeToString(authMsg.Nonce[:])).WithField(
	//"sig ", hex.EncodeToString(authMsg.Signature[:])).Trace("read msg")
	rPub, err := crypto.Ecrecover(authMsg.Nonce[:], authMsg.Signature[:])
	if err != nil {
		return nil, fmt.Errorf("sig invalid %v", err)
	}
	h := new(rawHandshake)
	rpub, err := importPublicKey(rPub[1:])
	if err != nil {
		log.WithError(err).Debug("handle authMsg failed")
		return nil, err
	}
	h.initNonce = authMsg.Nonce[:]
	h.remote = rpub

	h.respNonce = make([]byte, shaLen)
	if _, err = rand.Read(h.respNonce); err != nil {
		return nil, err
	}
	msg := new(RawHandshakeResponseMsg)
	copy(msg.Nonce[:], h.respNonce)
	msg.Version = 4
	buf := new(bytes.Buffer)
	b, err := msg.MarshalMsg(nil)
	if err != nil {
		return nil, err
	}
	buf.Write(b)
	enc, err := ecies.Encrypt(rand.Reader, h.remote, buf.Bytes(), nil, nil)
	if err != nil {
		log.WithError(err).Debug("enc failed")
		return nil, err
	}
	head := make([]byte, 3)
	putInt24(uint32(len(enc)), head)
	if _, err = conn.Write(head); err != nil {
		return nil, err
	}
	if _, err = conn.Write(enc); err != nil {
		return nil, err
	}
	return h.remote.ExportECDSA(), nil
}

// doEncHandshake runs the protocol handshake using authenticated
// messages. the protocol handshake is the first authenticated message
// and also verifies whether the encryption handshake 'worked' and the
// remote side actually provided the right public key.
func (t *rawTransport) doEncHandshake(prv *ecdsa.PrivateKey, dial *ecdsa.PublicKey) (*ecdsa.PublicKey, error) {
	var (
		pub *ecdsa.PublicKey
		err error
	)
	if dial == nil {
		pub, err = receiverRawHandshake(t.fd, prv)
	} else {
		err = initiatorRawencHandshake(t.fd, prv, dial)
		pub = dial
	}
	if err != nil {
		log.WithError(err).Debug("handshake error")
		return nil, err
	}
	t.wmu.Lock()
	t.rw = newRawFrameRW(t.fd)
	t.wmu.Unlock()
	return pub, nil
}

// rawFrameRW implements a simplified version of RLPx framing.
// chunked messages are not supported and all headers are equal to
// zeroHeader.
//
// rawFrameRW is not safe for concurrent use from multiple goroutines.
type rawFrameRW struct {
	conn   io.ReadWriter
	snappy bool
}

func newRawFrameRW(conn io.ReadWriter) *rawFrameRW {
	return &rawFrameRW{
		conn: conn,
	}
}

func (rw *rawFrameRW) WriteMsg(msg Msg) error {
	//ptype, _ := rlp.EncodeToBytes(msg.Code)
	ptype, _ := msg.Code.MarshalMsg(nil)
	var payload []byte
	// if snappy is enabled, compress message now
	if rw.snappy {
		if msg.Size > maxUint24 {
			return errPlainMessageTooLarge
		}
		payload, _ = msg.GetPayLoad()
		payload = snappy.Encode(nil, payload)

		msg.Payload = bytes.NewReader(payload)
		msg.Size = uint32(len(payload))
	} else {
		payload, _ = msg.GetPayLoad()
	}

	// write header
	headbuf := make([]byte, 6)
	fsize := uint32(len(ptype)) + msg.Size
	if fsize > maxUint24 {
		return errors.New("message size overflows uint24")
	}
	putInt24(fsize, headbuf) // TODO: check overflow
	copy(headbuf[3:], zeroHeader)
	// write header
	if _, err := rw.conn.Write(headbuf); err != nil {
		return err
	}
	if _, err := rw.conn.Write(ptype); err != nil {
		return err
	}
	if _, err := rw.conn.Write(payload); err != nil {
		return err
	}
	return nil
}

func (rw *rawFrameRW) ReadMsg() (msg Msg, err error) {
	// read the header
	headbuf := make([]byte, 6)
	if _, err := io.ReadFull(rw.conn, headbuf); err != nil {
		return msg, err
	}
	// verify header
	fsize := readInt24(headbuf)
	// ignore protocol type for now
	framebuf := make([]byte, fsize)
	if _, err := io.ReadFull(rw.conn, framebuf); err != nil {
		return msg, err
	}
	out, err := msg.Code.UnmarshalMsg(framebuf[:fsize])
	if err != nil {
		return msg, err
	}
	content := bytes.NewReader(out)
	msg.Size = uint32(content.Len())
	msg.Payload = content
	// if snappy is enabled, verify and decompress message
	if rw.snappy {
		payload, err := ioutil.ReadAll(msg.Payload)
		if err != nil {
			return msg, err
		}
		size, err := snappy.DecodedLen(payload)
		if err != nil {
			return msg, err
		}
		if size > int(maxUint24) {
			return msg, errPlainMessageTooLarge
		}
		payload, err = snappy.Decode(nil, payload)
		if err != nil {
			return msg, err
		}
		msg.Size, msg.Payload = uint32(size), bytes.NewReader(payload)
	}
	return msg, nil
}
