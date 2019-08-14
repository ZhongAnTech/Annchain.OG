package communicator

import (
	"bytes"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/message"
	"github.com/sirupsen/logrus"
)

// TrustfulPartnerCommunicator signs and validate messages using pubkey/privkey given by DKG/BLS
// It provides Trustful communication between partners with pubkeys
// All messages received from TrustfulPartnerCommunicator is considered crypto safe and sender verified.
type TrustfulPartnerCommunicator struct {
	incomingChannel   chan bft.BftMessage
	Signer            crypto.ISigner
	TermProvider      annsensus.TermProvider //TODO：not its job.
	P2PSender         P2PSender
	MyAccountProvider ConsensusAccountProvider
}

func NewTrustfulPeerCommunicator(signer crypto.ISigner, termProvider annsensus.TermProvider,
	myAccountProvider ConsensusAccountProvider, p2pSender P2PSender) *TrustfulPartnerCommunicator {
	return &TrustfulPartnerCommunicator{
		incomingChannel:   make(chan bft.BftMessage, 20),
		Signer:            signer,
		TermProvider:      termProvider,
		MyAccountProvider: myAccountProvider,
		P2PSender:         p2pSender,
	}
}

func (r *TrustfulPartnerCommunicator) Sign(msg bft.BftMessage) SignedOgParnterMessage {
	account := r.MyAccountProvider.Account()
	signed := SignedOgParnterMessage{
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
		r.P2PSender.AnonymousSendMessage(message.MessageTypeConsensus, &signed, &peer.PublicKey)
	}

}

// Unicast must be anonymous
func (r *TrustfulPartnerCommunicator) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	signed := r.Sign(msg)
	r.P2PSender.AnonymousSendMessage(message.MessageTypeConsensus, &signed, &peer.PublicKey)
}

// GetIncomingChannel provides a channel for downstream component consume the messages
// that are already verified by communicator
func (r *TrustfulPartnerCommunicator) GetIncomingChannel() chan bft.BftMessage {
	return r.incomingChannel
}

func (b *TrustfulPartnerCommunicator) VerifyParnterIdentity(publicKey crypto.PublicKey, sourcePartner int, termId uint32) bool {
	peers := b.TermProvider.Peers(termId)
	if sourcePartner < 0 || sourcePartner > len(peers)-1 {
		logrus.WithField("len partner ", len(peers)).WithField("sr ", sourcePartner).Warn("sourceId error")
		return false
	}
	partner := peers[sourcePartner]
	if bytes.Equal(partner.PublicKey.Bytes, publicKey.Bytes) {
		return true
	}
	logrus.Trace(publicKey.String(), " ", partner.PublicKey.String())
	return false

}

// handler for hub
func (b *TrustfulPartnerCommunicator) HandleIncomingMessage(msg *bft.BftMessage, peerId string) {
	switch msg.Type {
	case bft.BftMessageTypeProposal:
	case bft.BftMessageTypePreVote:
	case bft.BftMessageTypePreCommit:
	}
}
