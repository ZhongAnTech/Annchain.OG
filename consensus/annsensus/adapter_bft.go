package annsensus

import (
	"bytes"
	"errors"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/og/message"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/sirupsen/logrus"
)

// TrustfulBftAdapter signs and validate messages using pubkey/privkey given by DKG/BLS
type TrustfulBftAdapter struct {
	signatureProvider account.SignatureProvider
	termProvider      TermProvider
}

func (r *TrustfulBftAdapter) AdaptBftMessage(outgoingMsg *bft.BftMessage) (msg p2p_message.Message, err error) {
	signed := r.Sign(outgoingMsg)
	msg = &signed
	return
}

func NewTrustfulBftAdapter(
	signatureProvider account.SignatureProvider,
	termProvider TermProvider) *TrustfulBftAdapter {
	return &TrustfulBftAdapter{signatureProvider: signatureProvider, termProvider: termProvider}
}

func (r *TrustfulBftAdapter) Sign(msg *bft.BftMessage) message.SignedOgPartnerMessage {
	signed := message.SignedOgPartnerMessage{
		BftMessage: *msg,
		Signature:  r.signatureProvider.Sign(msg.Payload.SignatureTargets()),
		//SessionId:     partner.CurrentTerm(),
		//PublicKey: account.PublicKey.Bytes,
	}
	return signed
}

// Broadcast must be anonymous since it is actually among all partners, not all nodes.
//func (r *TrustfulBftAdapter) Broadcast(msg *bft.BftMessage, peers []bft.PeerInfo) {
//	signed := r.Sign(msg)
//	for _, peer := range peers {
//		r.p2pSender.AnonymousSendMessage(message.OGMessageType(msg.Type), &signed, &peer.PublicKey)
//	}
//}
//
//// Unicast must be anonymous
//func (r *TrustfulBftAdapter) Unicast(msg *bft.BftMessage, peer bft.PeerInfo) {
//	signed := r.Sign(msg)
//	r.p2pSender.AnonymousSendMessage(message.OGMessageType(msg.Type), &signed, &peer.PublicKey)
//}

func (b *TrustfulBftAdapter) VerifyParnterIdentity(signedMsg *message.SignedOgPartnerMessage) error {
	peers, err := b.termProvider.Peers(signedMsg.TermId)
	if err != nil {
		// this term is unknown.
		return err
	}
	// use public key to find sourcePartner
	for _, peer := range peers {
		if bytes.Equal(peer.PublicKey.Bytes, signedMsg.PublicKey) {
			return nil
		}
	}
	return errors.New("public key not found in current term")
}

func (b *TrustfulBftAdapter) VerifyMessageSignature(signedMsg *message.SignedOgPartnerMessage) error {
	ok := crypto.VerifySignature(signedMsg.PublicKey, signedMsg.Payload.SignatureTargets(), signedMsg.Signature)
	if !ok {
		return errors.New("signature invalid")
	}
	return nil
}

func (b *TrustfulBftAdapter) AdaptOgMessage(incomingMsg p2p_message.Message) (msg bft.BftMessage, err error) { // Only allows SignedOgPartnerMessage
	signedMsg, ok := incomingMsg.(*message.SignedOgPartnerMessage)
	if !ok {
		err = errors.New("message received is not a proper type for bft")
		return
	}
	err = b.VerifyParnterIdentity(signedMsg)
	if err != nil {
		logrus.WithField("term", signedMsg.TermId).WithError(err).Warn("bft message partner identity is not valid or unknown")
		err = errors.New("bft message partner identity is not valid or unknown")
		return
	}
	err = b.VerifyMessageSignature(signedMsg)
	if err != nil {
		logrus.WithError(err).Warn("bft message signature is not valid")
		err = errors.New("bft message signature is not valid")
		return
	}

	return signedMsg.BftMessage, nil
}

type PlainBftAdapter struct {
}

func (p PlainBftAdapter) AdaptOgMessage(incomingMsg p2p_message.Message) (msg bft.BftMessage, err error) {
	signedMsg, ok := incomingMsg.(*message.SignedOgPartnerMessage)
	if !ok {
		err = errors.New("message received is not a proper type for bft")
		return
	}
	msg = signedMsg.BftMessage
	return

}

func (p PlainBftAdapter) AdaptBftMessage(outgoingMsg *bft.BftMessage) (p2p_message.Message, error) {
	signed := message.SignedOgPartnerMessage{
		BftMessage: *outgoingMsg,
	}
	return &signed, nil
}
