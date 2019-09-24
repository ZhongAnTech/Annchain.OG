package communicator

import (
	"bytes"
	"errors"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/og/message"
	"github.com/sirupsen/logrus"
	"sync"
)

// TrustfulBftPartnerCommunicator signs and validate messages using pubkey/privkey given by DKG/BLS
// It provides Trustful communication between partners with pubkeys
// All messages received from TrustfulBftPartnerCommunicator is considered crypto safe and sender verified.
type TrustfulBftPartnerCommunicator struct {
	incomingChannel   chan bft.BftMessage     // the channel for subscribers to consume outside message.
	receivingChannel  chan *message.OGMessage // the channel for communicator to receive msg from outside
	SignatureProvider account.SignatureProvider
	TermProvider      annsensus.TermProvider // TODO：not its job.
	P2PSender         P2PSender              // upstream message sender

	quit   chan bool
	quitWg sync.WaitGroup
}

func (r *TrustfulBftPartnerCommunicator) GetReceivingChannel() chan *message.OGMessage {
	return r.receivingChannel
}

func (r *TrustfulBftPartnerCommunicator) Run() {
	// keep receiving OG messages and decrypt to incoming channel
	for {
		select {
		case <-r.quit:
			r.quitWg.Done()
			return
		case msg := <-r.receivingChannel:
			ogMsg := r.handleOgMessage(msg)
			if ogMsg == nil {
				continue
			}
			ffchan.NewTimeoutSenderShort(r.incomingChannel, ogMsg, "trustre")
			//r.incomingChannel <- ogMsg
		}
	}
}

func NewTrustfulPeerCommunicator(signatureProvider account.SignatureProvider, termProvider annsensus.TermProvider,
	p2pSender P2PSender) *TrustfulBftPartnerCommunicator {
	return &TrustfulBftPartnerCommunicator{
		incomingChannel:   make(chan bft.BftMessage, 20),
		SignatureProvider: signatureProvider,
		TermProvider:      termProvider,
		P2PSender:         p2pSender,
	}
}

func (r *TrustfulBftPartnerCommunicator) Sign(msg bft.BftMessage) message.SignedOgPartnerMessage {
	signed := message.SignedOgPartnerMessage{
		BftMessage: msg,
		Signature:  r.SignatureProvider.Sign(msg.Payload.SignatureTargets()),
		//SessionId:     partner.CurrentTerm(),
		//PublicKey: account.PublicKey.Bytes,
	}
	return signed
}

// Broadcast must be anonymous since it is actually among all partners, not all nodes.
func (r *TrustfulBftPartnerCommunicator) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
	signed := r.Sign(msg)
	for _, peer := range peers {
		r.P2PSender.AnonymousSendMessage(message.OGMessageType(msg.Type), &signed, &peer.PublicKey)
	}
}

// Unicast must be anonymous
func (r *TrustfulBftPartnerCommunicator) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	signed := r.Sign(msg)
	r.P2PSender.AnonymousSendMessage(message.OGMessageType(msg.Type), &signed, &peer.PublicKey)
}

// GetPipeOut provides a channel for downstream component consume the messages
// that are already verified by communicator
func (r *TrustfulBftPartnerCommunicator) GetPipeOut() chan bft.BftMessage {
	return r.incomingChannel
}

func (b *TrustfulBftPartnerCommunicator) VerifyParnterIdentity(signedMsg *message.SignedOgPartnerMessage) error {
	peers := b.TermProvider.Peers(signedMsg.TermId)
	// use public key to find sourcePartner
	for _, peer := range peers {
		if bytes.Equal(peer.PublicKey.Bytes, signedMsg.PublicKey) {
			return nil
		}
	}
	return errors.New("public key not found in current term")
}

func (b *TrustfulBftPartnerCommunicator) VerifyMessageSignature(signedMsg *message.SignedOgPartnerMessage) error {
	ok := crypto.VerifySignature(signedMsg.PublicKey, signedMsg.Payload.SignatureTargets(), signedMsg.Signature)
	if !ok {
		return errors.New("signature invalid")
	}
	return nil
}

// handler for hub or upstream annsensus module
func (r *TrustfulBftPartnerCommunicator) HandleIncomingMessage(msg bft.BftMessage) {
	r.receivingChannel <- msg
}

func (b *TrustfulBftPartnerCommunicator) AdaptOgMessage(incomingMsg *message.OGMessage) (msg bft.BftMessage, err error) {	// Only allows SignedOgPartnerMessage
	signedMsg, ok := incomingMsg.Message.(*message.SignedOgPartnerMessage)
	if !ok {
		err = errors.New("message received is not a proper type for bft")
		return
	}
	err := b.VerifyParnterIdentity(signedMsg)
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
