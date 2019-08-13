package communicator

import (
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/consensus/bft"
)

//go:generate msgp

// SignedOgParnterMessage is the message that is signed by partner.
// Consensus layer does not need to care about the signing. It is TrustfulPartnerCommunicator's job
//msgp:tuple SignedOgParnterMessage
type SignedOgParnterMessage struct {
	bft.BftMessage
	TermId     uint32
	ValidRound int
	//PublicKey  []byte
	Signature hexutil.Bytes
	PublicKey hexutil.Bytes
}
