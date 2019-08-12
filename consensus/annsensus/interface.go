package annsensus

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/consensus/bft"
)

// DkgTermProvider provide Dkg term that will be changed every term switching.
// DkgTermProvider maintains historic Peer info that can be retrieved by height.

type DkgTermProvider interface {
	CurrentDkgTerm() uint32
	Peers(height int) []bft.PeerInfo
	LatestPeers() []bft.PeerInfo
}

type AccountNonceProvider interface {
	GetNonce(account *account.Account) uint64
}
