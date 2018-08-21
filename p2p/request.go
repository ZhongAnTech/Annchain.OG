package p2p

import (
	"bufio"
	"context"
	"fmt"
	"log"

	"github.com/golang/protobuf/proto"

	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
)

// RequestProtocol type
type RequestProtocol struct {
	node *Node
}

const dataRequestProtocol = "/dataRequest/1.0.0"

// NewRequestProtocol defines the request protocol, which allows others to query data
func NewRequestProtocol(node *Node) *RequestProtocol {
	p := &RequestProtocol{
		node: node,
	}
	node.SetStreamHandler(dataRequestProtocol, p.onDataRequest)
	return p
}

// reads a protobuf go data object from a network stream
// data: reference of protobuf go data object(not the object itself)
// s: network stream to read the data from
func readProtoMessage(data proto.Message, s inet.Stream) bool {
	/*
		decoder := protobufCodec.Multicodec(nil).Decoder(bufio.NewReader(s))
		err := decoder.Decode(data)
		if err != nil {
			log.Println("readProtoMessage: ", err)
			return false
		}
	*/
	return true
}

// helper method - writes a protobuf go data object to a network stream
// data: reference of protobuf go data object to send (not the object itself)
// s: network stream to write the data to
func sendProtoMessage(data proto.Message, s inet.Stream) bool {
	/*
		writer := bufio.NewWriter(s)
		enc := protobufCodec.Multicodec(nil).Encoder(writer)
		err := enc.Encode(data)
		if err != nil {
			log.Println(err)
			return false
		}
		writer.Flush()
	*/
	return true
}

func sendBytes(data []byte, s inet.Stream) bool {
	writer := bufio.NewWriter(s)
	_, err := writer.Write(data)
	if err != nil {
		log.Println(err)
		return false
	}
	writer.Flush()
	return true
}

func readBytes(data []byte, s inet.Stream) bool {
	reader := bufio.NewReader(s)
	_, err := reader.Read(data)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (p *RequestProtocol) onDataRequest(s inet.Stream) {
	req := makedataRequest()
	if !readBytes(req, s) {
		s.Close()
		return
	}
	res := makeDataResponse()
	if !sendBytes(res, s) {
		log.Printf("onShardPeerRequest: failed to send proto message %v", res)
		s.Close()
	}
}

func (p *RequestProtocol) RequestData(ctx context.Context, peerID peer.ID) (resData []byte, e error) {
	s, err := p.node.NewStream(
		ctx,
		peerID,
		"data/1.0.0",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open new stream %v", err)
	}
	req := makedataRequest()
	if !sendBytes(req, s) {
		return nil, fmt.Errorf("failed to send request")
	}
	data := makeDataResponse()
	if !readBytes(data, s) {
		return nil, fmt.Errorf("failed to read response proto")
	}
	return resData, nil
}

func makedataRequest() []byte {
	return nil

}

func makeDataResponse() []byte {
	return nil

}
