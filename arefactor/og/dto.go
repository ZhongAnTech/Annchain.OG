package og

import (
	"github.com/libp2p/go-libp2p-core/crypto"
)

type PeerMember struct {
	PeerId    string        // node peer id to connect to peers
	PublicKey crypto.PubKey // account public key to verify messages
}

type Committee struct {
	Peers   []*PeerMember
	Version int
}
