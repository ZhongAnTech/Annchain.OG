package annsensus

import (
	"bytes"
	"errors"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/og/communicator"
	"github.com/annchain/OG/og/message"
	"github.com/sirupsen/logrus"
)

// TrustfulBftAdapter signs and validate messages using pubkey/privkey given by DKG/BLS
// It provides Trustful communication between partners with pubkeys
// All messages received from TrustfulBftAdapter is considered crypto safe and sender verified.
type TrustfulBftAdapter struct {
	signatureProvider account.SignatureProvider
	termProvider      TermProvider // TODO：not its job.
	p2pSender         communicator.P2PSender
}

func NewTrustfulBftAdapter(signatureProvider account.SignatureProvider, termProvider TermProvider, p2pSender communicator.P2PSender) *TrustfulBftAdapter {
	return &TrustfulBftAdapter{signatureProvider: signatureProvider, termProvider: termProvider, p2pSender: p2pSender}
}

func (r *TrustfulBftAdapter) Sign(msg bft.BftMessage) message.SignedOgPartnerMessage {
	signed := message.SignedOgPartnerMessage{
		BftMessage: msg,
		Signature:  r.signatureProvider.Sign(msg.Payload.SignatureTargets()),
		//SessionId:     partner.CurrentTerm(),
		//PublicKey: account.PublicKey.Bytes,
	}
	return signed
}

// Broadcast must be anonymous since it is actually among all partners, not all nodes.
func (r *TrustfulBftAdapter) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
	signed := r.Sign(msg)
	for _, peer := range peers {
		r.p2pSender.AnonymousSendMessage(message.OGMessageType(msg.Type), &signed, &peer.PublicKey)
	}
}

// Unicast must be anonymous
func (r *TrustfulBftAdapter) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	signed := r.Sign(msg)
	r.p2pSender.AnonymousSendMessage(message.OGMessageType(msg.Type), &signed, &peer.PublicKey)
}

func (b *TrustfulBftAdapter) VerifyParnterIdentity(signedMsg *message.SignedOgPartnerMessage) error {
	peers := b.TermProvider.Peers(signedMsg.TermId)
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

func (b *TrustfulBftAdapter) AdaptOgMessage(incomingMsg *message.OGMessage) (msg bft.BftMessage, err error) { // Only allows SignedOgPartnerMessage
	signedMsg, ok := incomingMsg.Message.(*message.SignedOgPartnerMessage)
	if !ok {
		err = errors.New("message received is not a proper type for bft")
		return
	}
	err = b.VerifyParnterIdentity(signedMsg)
	if err != nil {
		logrus.WithField("term", signedMsg.TermId).WithError(err).Warn("bft message partner identity is not valid")
		err = errors.New("bft message partner identity is not valid")
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
