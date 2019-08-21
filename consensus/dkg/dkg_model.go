package dkg

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/kyber/v3"
)

type PeerInfo struct {
	Id             int
	PublicKey      crypto.PublicKey `json:"-"`
	Address        common.Address   `json:"address"`
	PublicKeyBytes hexutil.Bytes    `json:"public_key"`
}

type PartPub struct {
	Point kyber.Point
	Peer  PeerInfo
}

type PartSec struct {
	PartPub
	Scalar     kyber.Scalar
	PrivateKey crypto.PrivateKey
}

type PartPubs []PartPub

func (p PartPubs) Points() []kyber.Point {
	var points []kyber.Point
	for _, v := range p {
		points = append(points, v.Point)
	}
	return points
}

func (p PartPubs) Peers() []PeerInfo {
	var peers []PeerInfo
	for _, v := range p {
		peers = append(peers, v.Peer)
	}
	return peers
}

func (h PartPubs) Len() int {
	return len(h)
}

//
//func (h PartPubs) Less(i, j int) bool {
//	return h[i].PublicKey.String() < h[j].PublicKey.String()
//}
//
//func (h PartPubs) Swap(i, j int) {
//	h[i], h[j] = h[j], h[i]
//}
//
//type PeerInfo struct {
//	MyIndex        int
//	PublicKey crypto.PublicKey `json:"-"`
//
//	Part kyber.Point
//}
