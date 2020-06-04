package og

import "crypto"

type PeerMember struct {
	PeerId    string           // node peer id to connect to peers
	PublicKey crypto.PublicKey // account public key to verify messages
}

type Committee struct {
	Peers   []*PeerMember
	Version int
}
