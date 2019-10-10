package annsensus

import (
	"errors"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/types/p2p_message"
)

// TrustfulDkgAdapter signs and validate messages using pubkey/privkey given by DKG/BLS
type TrustfulDkgAdapter struct {
	signatureProvider account.SignatureProvider
	termProvider      TermProvider
}

func (r *TrustfulDkgAdapter) AdaptDkgMessage(outgoingMsg *dkg.DkgMessage) (msg p2p_message.Message, err error) {
	return
}

func NewTrustfulDkgAdapter() *TrustfulDkgAdapter {
	return &TrustfulDkgAdapter{}
}

func (b *TrustfulDkgAdapter) AdaptOgMessage(incomingMsg p2p_message.Message) (msg dkg.DkgMessage, err error) { // Only allows SignedOgPartnerMessage
	return
}

type PlainDkgAdapter struct {
}

func (p PlainDkgAdapter) AdaptOgMessage(incomingMsg p2p_message.Message) (msg dkg.DkgMessage, err error) {
	f, ok := incomingMsg.(*dkg.DkgMessage)
	if !ok {
		err = errors.New("cannot convert p2pmessage to dkgmessage")
		return
	}
	return *f, nil

}

func (p PlainDkgAdapter) AdaptDkgMessage(outgoingMsg *dkg.DkgMessage) (p2p_message.Message, error) {
	return outgoingMsg, nil
}
