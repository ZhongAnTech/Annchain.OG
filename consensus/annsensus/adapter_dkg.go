package annsensus

import (
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/types/p2p_message"
)

// DkgAdapter signs and validate messages using pubkey/privkey given by DKG/BLS
type DkgAdapter struct {
	signatureProvider account.SignatureProvider
	termProvider      TermProvider
}

func (r *DkgAdapter) AdaptDkgMessage(outgoingMsg *dkg.DkgMessage) (msg p2p_message.Message, err error) {
	return
}

func NewDkgAdapter() *DkgAdapter {
	return &DkgAdapter{}
}

func (b *DkgAdapter) AdaptOgMessage(incomingMsg p2p_message.Message) (msg dkg.DkgMessage, err error) { // Only allows SignedOgPartnerMessage
	return
}
