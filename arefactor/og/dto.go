package og

import "github.com/annchain/OG/arefactor/ogcrypto_interface"

type PeerMember struct {
	PeerId    string                       // node peer id to connect to peers
	PublicKey ogcrypto_interface.PublicKey // account public key to verify messages
}

type Committee struct {
	Peers   []*PeerMember
	Version int
}
