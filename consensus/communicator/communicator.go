package communicator

import (
	"bytes"
	"errors"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/message"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/sirupsen/logrus"
)

// TrustfulPartnerCommunicator signs and validate messages using pubkey/privkey given by DKG/BLS
// It provides Trustful communication between partners with pubkeys
// All messages received from TrustfulPartnerCommunicator is considered crypto safe and sender verified.
type TrustfulPartnerCommunicator struct {
	incomingChannel   chan *message.SignedOgPartnerMessage
	Signer            crypto.ISigner
	TermProvider      annsensus.TermProvider //TODO：not its job.
	P2PSender         P2PSender
	MyAccountProvider ConsensusAccountProvider
}

func NewTrustfulPeerCommunicator(signer crypto.ISigner, termProvider annsensus.TermProvider,
	myAccountProvider ConsensusAccountProvider, p2pSender P2PSender) *TrustfulPartnerCommunicator {
	return &TrustfulPartnerCommunicator{
		incomingChannel:   make(chan *message.SignedOgPartnerMessage, 20),
		Signer:            signer,
		TermProvider:      termProvider,
		MyAccountProvider: myAccountProvider,
		P2PSender:         p2pSender,
	}
}

func (r *TrustfulPartnerCommunicator) Sign(msg bft.BftMessage) message.SignedOgPartnerMessage {
	account := r.MyAccountProvider.Account()
	signed := message.SignedOgPartnerMessage{
		BftMessage: msg,
		Signature:  r.Signer.Sign(account.PrivateKey, msg.Payload.SignatureTargets()).Bytes,
		//TermId:     partner.CurrentTerm(),
		PublicKey: account.PublicKey.Bytes,
	}
	return signed
}

// Broadcast must be anonymous since it is actually among all partners, not all nodes.
func (r *TrustfulPartnerCommunicator) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
	signed := r.Sign(msg)
	for _, peer := range peers {
		r.P2PSender.AnonymousSendMessage(message.OGMessageType(msg.Type), &signed, &peer.PublicKey)
	}

}

// Unicast must be anonymous
func (r *TrustfulPartnerCommunicator) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	signed := r.Sign(msg)
	r.P2PSender.AnonymousSendMessage(message.OGMessageType(msg.Type), &signed, &peer.PublicKey)
}

// GetIncomingChannel provides a channel for downstream component consume the messages
// that are already verified by communicator
func (r *TrustfulPartnerCommunicator) GetIncomingChannel() chan *message.SignedOgPartnerMessage {
	return r.incomingChannel
}

func (b *TrustfulPartnerCommunicator) VerifyParnterIdentity(signedMsg *message.SignedOgPartnerMessage) error {
	peers := b.TermProvider.Peers(signedMsg.TermId)
	// use public key to find sourcePartner
	for _, peer := range peers {
		if bytes.Equal(peer.PublicKey.Bytes, signedMsg.PublicKey) {
			return nil
		}
	}
	return errors.New("public key not found in current term")
}

func (b *TrustfulPartnerCommunicator) VerifyMessageSignature(signedMsg *message.SignedOgPartnerMessage) error {
	ok := crypto.VerifySignature(signedMsg.PublicKey, signedMsg.Payload.SignatureTargets(), signedMsg.Signature)
	if !ok {
		return errors.New("signature invalid")
	}
	return nil
}

// handler for hub
func (b *TrustfulPartnerCommunicator) HandleIncomingMessage(msg p2p_message.Message) {
	// Only allows SignedOgPartnerMessage
	signedMsg, ok := msg.(*message.SignedOgPartnerMessage)
	if !ok {
		logrus.Warn("message received is not a proper type for bft")
		return
	}
	err := b.VerifyParnterIdentity(signedMsg)
	if err != nil {
		logrus.WithField("term", signedMsg.TermId).WithError(err).Warn("bft message partner identity is not valid")
		return
	}
	err = b.VerifyMessageSignature(signedMsg)
	if err != nil {
		logrus.WithError(err).Warn("bft message signature is not valid")
		return
	}

	b.incomingChannel <- signedMsg
}
