package annsensus

import (
	"bytes"
	"errors"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/account"
	"github.com/sirupsen/logrus"
)

type ProxyBftPeerCommunicator struct {
	bftMessageAdapter BftMessageAdapter // either TrustfulBftAdapter or PlainBftAdapter
	annsensusOutgoing AnnsensusPeerCommunicatorOutgoing
	pipe              chan *bft.BftMessageEvent
}

func NewProxyBftPeerCommunicator(
	bftMessageAdapter BftMessageAdapter,
	annsensusOutgoing AnnsensusPeerCommunicatorOutgoing) *ProxyBftPeerCommunicator {
	return &ProxyBftPeerCommunicator{
		bftMessageAdapter: bftMessageAdapter,
		annsensusOutgoing: annsensusOutgoing,
		pipe:              make(chan *bft.BftMessageEvent),
	}
}

func (p *ProxyBftPeerCommunicator) Broadcast(msg bft.BftMessage, peers []bft.BftPeer) {
	annsensusMessage, err := p.bftMessageAdapter.AdaptBftMessage(msg)
	if err != nil {
		panic("adapt should never fail")
	}
	// adapt the interface so that the request can be handled by annsensus
	annsensusPeers := make([]AnnsensusPeer, len(peers))
	for i, peer := range peers {
		adaptedValue, err := p.bftMessageAdapter.AdaptBftPeer(peer)
		if err != nil {
			panic("adapt should never fail")
		}
		annsensusPeers[i] = adaptedValue
	}

	p.annsensusOutgoing.Broadcast(annsensusMessage, annsensusPeers)
}

func (p *ProxyBftPeerCommunicator) Unicast(msg bft.BftMessage, peer bft.BftPeer) {
	// adapt the interface so that the request can be handled by annsensus
	annsensusMessage, err := p.bftMessageAdapter.AdaptBftMessage(msg)
	if err != nil {
		panic("adapt should never fail")
	}
	annsensusPeer, err := p.bftMessageAdapter.AdaptBftPeer(peer)
	if err != nil {
		panic("adapt should never fail")
	}
	p.annsensusOutgoing.Unicast(annsensusMessage, annsensusPeer)
}

func (p *ProxyBftPeerCommunicator) GetPipeOut() chan *bft.BftMessageEvent {
	// the channel to be consumed by the downstream.
	return p.pipe
}

func (p *ProxyBftPeerCommunicator) GetPipeIn() chan *bft.BftMessageEvent {
	// the channel to be fed by other peers
	return p.pipe
}

func (p *ProxyBftPeerCommunicator) Run() {
	// nothing to do
	return
}

type BftMessageUnmarshaller struct {
}

func (b *BftMessageUnmarshaller) Unmarshal(messageType bft.BftMessageType, message []byte) (outMsg bft.BftMessage, err error) {
	switch messageType {
	case bft.BftMessageTypeProposal:
		m := &bft.BftMessageProposal{
			Value: &bft.StringProposal{},
		}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case bft.BftMessageTypePreVote:
		m := &bft.BftMessagePreVote{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case bft.BftMessageTypePreCommit:
		m := &bft.BftMessagePreCommit{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	default:
		err = errors.New("message type of Bft not supported")
	}
	return
}

// TrustfulBftAdapter signs and validate messages using pubkey/privkey given by DKG/BLS
type TrustfulBftAdapter struct {
	signatureProvider      account.SignatureProvider
	termHolder             HistoricalTermsHolder
	bftMessageUnmarshaller *BftMessageUnmarshaller
}

func (r *TrustfulBftAdapter) AdaptAnnsensusPeer(annPeer AnnsensusPeer) (bft.BftPeer, error) {
	return bft.BftPeer{
		Id:             annPeer.Id,
		PublicKey:      annPeer.PublicKey,
		Address:        annPeer.Address,
		PublicKeyBytes: annPeer.PublicKeyBytes,
	}, nil
}

func (r *TrustfulBftAdapter) AdaptBftPeer(bftPeer bft.BftPeer) (AnnsensusPeer, error) {
	return AnnsensusPeer{
		Id:             bftPeer.Id,
		PublicKey:      bftPeer.PublicKey,
		Address:        bftPeer.Address,
		PublicKeyBytes: bftPeer.PublicKeyBytes,
	}, nil
}

func (r *TrustfulBftAdapter) AdaptBftMessage(outgoingMsg bft.BftMessage) (msg AnnsensusMessage, err error) {
	signed := r.Sign(outgoingMsg)
	msg = &signed
	return
}

func NewTrustfulBftAdapter(
	signatureProvider account.SignatureProvider,
	termHolder HistoricalTermsHolder) *TrustfulBftAdapter {
	return &TrustfulBftAdapter{signatureProvider: signatureProvider, termHolder: termHolder}
}

func (r *TrustfulBftAdapter) Sign(rawMessage bft.BftMessage) AnnsensusMessageBftSigned {
	publicKey, signature := r.signatureProvider.Sign(rawMessage.SignatureTargets())
	signedMessage := AnnsensusMessageBftSigned{
		InnerMessageType: uint16(rawMessage.GetType()),
		InnerMessage:     rawMessage.SignatureTargets(),
		Signature:        signature,
		PublicKey:        publicKey,
	}
	//SessionId:     partner.CurrentTerm(),
	//PublicKey: account.PublicKey.KeyBytes,
	return signedMessage
}

// Multicast must be anonymous since it is actually among all partners, not all nodes.
//func (r *TrustfulBftAdapter) Multicast(msg bft.BftMessage, peers []bft.BftPeer) {
//	signed := r.Sign(msg)
//	for _, peer := range peers {
//		r.p2pSender.AnonymousSendMessage(message.BinaryMessageType(msg.Type), &signed, &peer.PublicKey)
//	}
//}
//
//// Unicast must be anonymous
//func (r *TrustfulBftAdapter) Unicast(msg bft.BftMessage, peer bft.BftPeer) {
//	signed := r.Sign(msg)
//	r.p2pSender.AnonymousSendMessage(message.BinaryMessageType(msg.Type), &signed, &peer.PublicKey)
//}

func (b *TrustfulBftAdapter) VerifyParnterIdentity(signedMsg *AnnsensusMessageBftSigned) error {
	term, ok := b.termHolder.GetTermById(signedMsg.TermId)
	if !ok {
		// this term is unknown.
		return errors.New("term not found")
	}

	// use public key to find sourcePartner
	for _, peer := range term.contextProvider.GetTerm().Senators {
		if bytes.Equal(peer.PublicKey.KeyBytes, signedMsg.PublicKey) {
			return nil
		}
	}
	return errors.New("public key not found in current term")
}

func (b *TrustfulBftAdapter) VerifyMessageSignature(outMsg bft.BftMessage, publicKey []byte, signature []byte) error {
	ok := crypto.VerifySignature(publicKey, outMsg.SignatureTargets(), signature)
	if !ok {
		return errors.New("signature invalid")
	}
	return nil
}

func (b *TrustfulBftAdapter) AdaptAnnsensusMessage(incomingMsg AnnsensusMessage) (msg bft.BftMessage, err error) { // Only allows SignedOgPartnerMessage
	if incomingMsg.GetType() != AnnsensusMessageTypeBftSigned {
		err = errors.New("TrustfulBftAdapter received a message of an unsupported type")
		return
	}

	signedMsg, ok := incomingMsg.(*AnnsensusMessageBftSigned)
	if !ok {
		err = errors.New("TrustfulBftAdapter received a message of type AnnsensusMessageBftSigned but it is not.")
		return
	}

	// check inner type
	bftMessage, err := b.bftMessageUnmarshaller.Unmarshal(bft.BftMessageType(signedMsg.InnerMessageType), signedMsg.InnerMessage)
	if err != nil {
		return
	}

	err = b.VerifyParnterIdentity(signedMsg)
	if err != nil {
		logrus.WithField("term", signedMsg.TermId).WithError(err).Warn("bft message partner identity is not valid or unknown")
		err = errors.New("bft message partner identity is not valid or unknown")
		return
	}

	err = b.VerifyMessageSignature(bftMessage, signedMsg.PublicKey, signedMsg.Signature)
	if err != nil {
		logrus.WithError(err).Warn("bft message signature is not valid")
		err = errors.New("bft message signature is not valid")
		return
	}

	return bftMessage, nil
}

// PlainBftAdapter will not wrap the message using MessageTypeAnnsensusSigned
type PlainBftAdapter struct {
	BftMessageUnmarshaller *BftMessageUnmarshaller
}

func (p PlainBftAdapter) AdaptAnnsensusPeer(annPeer AnnsensusPeer) (bft.BftPeer, error) {
	return bft.BftPeer{
		Id:             annPeer.Id,
		PublicKey:      annPeer.PublicKey,
		Address:        annPeer.Address,
		PublicKeyBytes: annPeer.PublicKeyBytes,
	}, nil
}

func (p PlainBftAdapter) AdaptBftPeer(bftPeer bft.BftPeer) (AnnsensusPeer, error) {
	return AnnsensusPeer{
		Id:             bftPeer.Id,
		PublicKey:      bftPeer.PublicKey,
		Address:        bftPeer.Address,
		PublicKeyBytes: bftPeer.PublicKeyBytes,
	}, nil
}

func (p PlainBftAdapter) AdaptAnnsensusMessage(incomingMsg AnnsensusMessage) (msg bft.BftMessage, err error) {
	if incomingMsg.GetType() != AnnsensusMessageTypeBftPlain {
		err = errors.New("PlainBftAdapter received a message of an unsupported type")
		return
	}
	iMsg := incomingMsg.(*AnnsensusMessageBftPlain)
	innerMessageType := bft.BftMessageType(iMsg.InnerMessageType)

	switch innerMessageType {
	case bft.BftMessageTypeProposal:
		fallthrough
	case bft.BftMessageTypePreVote:
		fallthrough
	case bft.BftMessageTypePreCommit:
		msg, err = p.BftMessageUnmarshaller.Unmarshal(innerMessageType, iMsg.InnerMessage)
	default:
		err = errors.New("PlainBftAdapter received a message of an unsupported inner type")
	}
	return

}

func (p PlainBftAdapter) AdaptBftMessage(outgoingMsg bft.BftMessage) (adaptedMessage AnnsensusMessage, err error) {
	var msgBytes []byte
	switch outgoingMsg.GetType() {
	case bft.BftMessageTypeProposal:
		omsg := outgoingMsg.(*bft.BftMessageProposal)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = &AnnsensusMessageBftPlain{
			InnerMessageType: uint16(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case bft.BftMessageTypePreVote:
		omsg := outgoingMsg.(*bft.BftMessagePreVote)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = &AnnsensusMessageBftPlain{
			InnerMessageType: uint16(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case bft.BftMessageTypePreCommit:
		omsg := outgoingMsg.(*bft.BftMessagePreCommit)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = &AnnsensusMessageBftPlain{
			InnerMessageType: uint16(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	default:
		err = errors.New("PlainBftAdapter received a message of an unsupported type")
	}
	return
}
