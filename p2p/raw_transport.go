package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/ecies"
	"github.com/golang/snappy"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"time"
)

//go:generate msgp
type RawTransport struct {
	fd net.Conn
	MsgReadWriter
	rmu, wmu sync.Mutex
	rw       *rawFrameRW
}

func newRawTransport(fd net.Conn) transport {
	fd.SetDeadline(time.Now().Add(handshakeTimeout))
	return &RawTransport{fd: fd}
}

func (t *RawTransport) ReadMsg() (Msg, error) {
	t.rmu.Lock()
	defer t.rmu.Unlock()
	t.fd.SetReadDeadline(time.Now().Add(frameReadTimeout))
	return t.rw.ReadMsg()
}

func (t *RawTransport) WriteMsg(msg Msg) error {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	t.fd.SetWriteDeadline(time.Now().Add(frameWriteTimeout))
	return t.rw.WriteMsg(msg)
}

func (t *RawTransport) close(err error) {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	// Tell the remote end why we're disconnecting if possible.
	if t.rw != nil {
		if r, ok := err.(DiscReason); ok && r != DiscNetworkError {
			// RawTransport tries to send DiscReason to disconnected peer
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

func (t *RawTransport) doProtoHandshake(our *ProtoHandshake) (their *ProtoHandshake, err error) {
	// Writing our handshake happens concurrently, we prefer
	// returning the handshake read error. If the remote side
	// disconnects us early with a valid reason, we should return it
	// as the error so it can be tracked elsewhere.
	werr := make(chan error, 1)
	b, _ := our.MarshalMsg(nil)
	go func() { werr <- Send(t.rw, handshakeMsg, b) }()
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
	Signature       [sigLen]byte
	InitiatorPubkey [pubLen]byte
	Nonce           [shaLen]byte
	Version         uint
}

// RLPx v4 handshake response (defined in EIP-8).
type RawHandshakeResponseMsg struct {
	RemotePubkey [pubLen]byte
	Nonce        [shaLen]byte
	Signature       [sigLen]byte
	Version      uint
}

// rawHandshake contains the state of the encryption handshake.
type rawHandshake struct {
	initiator            bool
	remote               *ecies.PublicKey  // remote-pubk
	initNonce, respNonce []byte            // nonce
}


func (h *rawHandshake) handleAuthMsg(msg *RawHandshakeMsg, prv *ecdsa.PrivateKey) error {
	// Import the remote identity.
	rpub, err := importPublicKey(msg.InitiatorPubkey[:])
	if err != nil {
		return err
	}
	h.initNonce = msg.Nonce[:]
	h.remote = rpub

	// Check the signature.
	return nil
}


func initiatorRawcHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, remote *ecdsa.PublicKey) (err error) {
	h := &rawHandshake{initiator: true, remote: ecies.ImportECDSAPublic(remote)}
	h.initNonce = make([]byte, shaLen)
	_, err = rand.Read(h.initNonce)
	if err != nil {
		return  err
	}
	msg := &RawHandshakeMsg{
		Version:1,
	}
	copy(msg.Nonce[:], h.initNonce)
	signature, err := crypto.Sign(h.initNonce, prv)
	if err != nil {
		return  err
	}
	copy(msg.Signature[:], signature)
	copy(msg.InitiatorPubkey[:], crypto.FromECDSAPub(&prv.PublicKey)[1:])
	copy(msg.Nonce[:], h.initNonce)
	buf := new(bytes.Buffer)
	b, err := msg.MarshalMsg(nil)
	if err != nil {
		return err
	}
	buf.Write(b)
	enc, err := ecies.Encrypt(rand.Reader, h.remote, buf.Bytes(), nil, nil)
	if err!=nil {
		return  err
	}
	if _, err = conn.Write(enc); err != nil {
		return err
	}
	authRespMsg := new(RawHandshakeResponseMsg)
    err = readRawHandshakeMsgResp(authRespMsg, len(enc), prv, conn)
	if err != nil {
		return err
	}
	if !crypto.VerifySignature(authRespMsg.RemotePubkey[:],authRespMsg.Nonce[:] ,authRespMsg.Signature[:]) {
		return  fmt.Errorf("sig invalid")
	}
	h.respNonce = authRespMsg.Nonce[:]
	h.remote, err = importPublicKey(authRespMsg.RemotePubkey[:])
	return nil
}

func readRawHandshakeMsgResp(msg *RawHandshakeResponseMsg, plainSize int, prv *ecdsa.PrivateKey, r io.Reader) (  error) {
	buf := make([]byte, plainSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return  err
	}
	// Attempt decoding pre-EIP-8 "plain" format.
	key := ecies.ImportECDSA(prv)
	dec, err := key.Decrypt(buf, nil, nil);
	if  err != nil {
	 return err
	}
	_,err = msg.UnmarshalMsg(dec)
	//	fmt.Println("is plain",msg)
	return  nil
}

func readRawHandshakeMsg(msg *RawHandshakeMsg, plainSize int, prv *ecdsa.PrivateKey, r io.Reader) (  error) {
	buf := make([]byte, plainSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return  err
	}
	// Attempt decoding pre-EIP-8 "plain" format.
	key := ecies.ImportECDSA(prv)
	dec, err := key.Decrypt(buf, nil, nil)
	if  err != nil {
		return err
	}
	_,err = msg.UnmarshalMsg(dec)
	//	fmt.Println("is plain",msg)
	return  nil
}


func receiverRawHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey) (*ecdsa.PublicKey ,error) {
	authMsg := new(RawHandshakeMsg)
	err := readRawHandshakeMsg(authMsg, encAuthMsgLen, prv, conn)
	if err != nil {
		return nil , err
	}
	if !crypto.VerifySignature(authMsg.InitiatorPubkey[:],authMsg.Nonce[:] ,authMsg.Signature[:]) {
		return nil , fmt.Errorf("sig invalid")
	}
	h := new(rawHandshake)
	if err := h.handleAuthMsg(authMsg, prv); err != nil {
		return nil , err
	}
	h.respNonce = make([]byte, shaLen)
	if _, err = rand.Read(h.respNonce); err != nil {
		return nil , err
	}
	msg := new(RawHandshakeResponseMsg)
	copy(msg.Nonce[:], h.respNonce)
	copy(msg.RemotePubkey[:], crypto.FromECDSAPub(&prv.PublicKey)[1:])
	msg.Version = 4
	buf := new(bytes.Buffer)
	b, err := msg.MarshalMsg(nil)
	if err != nil {
		return nil , err
	}
	buf.Write(b)
	enc, err := ecies.Encrypt(rand.Reader, h.remote, buf.Bytes(), nil, nil)
	if err!=nil {
		return nil , err
	}
	if _, err = conn.Write(enc); err != nil {
		return nil , err
	}
	return h.remote.ExportECDSA(), nil
}

// doEncHandshake runs the protocol handshake using authenticated
// messages. the protocol handshake is the first authenticated message
// and also verifies whether the encryption handshake 'worked' and the
// remote side actually provided the right public key.
func (t *RawTransport) doEncHandshake(prv *ecdsa.PrivateKey, dial *ecdsa.PublicKey) (*ecdsa.PublicKey, error) {
	var (
		pub *ecdsa.PublicKey
		err error
	)
	if dial == nil {
		pub, err = receiverRawHandshake(t.fd, prv)
	} else {
		err = initiatorRawcHandshake(t.fd, prv, dial)
		pub = dial
	}
	if err != nil {
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
	conn io.ReadWriter
	snappy bool
}

func newRawFrameRW(conn io.ReadWriter) *rawFrameRW {
	return &rawFrameRW{
		conn:       conn,
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

