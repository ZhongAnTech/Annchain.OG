package annsensus

import (
	"bytes"
	"errors"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/og/protocol_message"
	"github.com/annchain/OG/types/general_message"
	"github.com/sirupsen/logrus"
)

type BftMessageUnmarshaller struct {
}

func (b *BftMessageUnmarshaller) Unmarshal(messageType general_message.BinaryMessageType, message []byte) (outMsg bft.BftMessage, err error) {
	switch bft.BftMessageType(messageType) {
	case bft.BftMessageTypeProposal:
		m := &bft.MessageProposal{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case bft.BftMessageTypePreVote:
		m := &bft.MessagePreVote{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case bft.BftMessageTypePreCommit:
		m := &bft.MessagePreCommit{}
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
	termProvider           TermProvider
	bftMessageUnmarshaller *BftMessageUnmarshaller
}

func (r *TrustfulBftAdapter) AdaptBftMessage(outgoingMsg bft.BftMessage) (msg general_message.TransportableMessage, err error) {
	signed := r.Sign(outgoingMsg)
	msg = &signed
	return
}

func NewTrustfulBftAdapter(
	signatureProvider account.SignatureProvider,
	termProvider TermProvider) *TrustfulBftAdapter {
	return &TrustfulBftAdapter{signatureProvider: signatureProvider, termProvider: termProvider}
}

func (r *TrustfulBftAdapter) Sign(msg bft.BftMessage) protocol_message.MessageSigned {
	publicKey, signature := r.signatureProvider.Sign(msg.SignatureTargets())
	signed := protocol_message.MessageSigned{
		InnerMessageType: general_message.BinaryMessageType(msg.GetType()),
		InnerMessage:     msg.SignatureTargets(),
		Signature:        signature,
		PublicKey:        publicKey,
	}
	//SessionId:     partner.CurrentTerm(),
	//PublicKey: account.PublicKey.Bytes,
	return signed
}

// Broadcast must be anonymous since it is actually among all partners, not all nodes.
//func (r *TrustfulBftAdapter) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
//	signed := r.Sign(msg)
//	for _, peer := range peers {
//		r.p2pSender.AnonymousSendMessage(message.BinaryMessageType(msg.Type), &signed, &peer.PublicKey)
//	}
//}
//
//// Unicast must be anonymous
//func (r *TrustfulBftAdapter) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
//	signed := r.Sign(msg)
//	r.p2pSender.AnonymousSendMessage(message.BinaryMessageType(msg.Type), &signed, &peer.PublicKey)
//}

func (b *TrustfulBftAdapter) VerifyParnterIdentity(signedMsg *protocol_message.MessageSigned) error {
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

func (b *TrustfulBftAdapter) VerifyMessageSignature(outMsg bft.BftMessage, publicKey []byte, signature []byte) error {
	ok := crypto.VerifySignature(publicKey, outMsg.SignatureTargets(), signature)
	if !ok {
		return errors.New("signature invalid")
	}
	return nil
}

func (b *TrustfulBftAdapter) AdaptOgMessage(incomingMsg general_message.TransportableMessage) (msg bft.BftMessage, err error) { // Only allows SignedOgPartnerMessage
	if incomingMsg.GetType() != general_message.MessageTypeSigned {
		err = errors.New("TrustfulBftAdapter received a message of an unsupported type")
		return
	}

	signedMsg, ok := incomingMsg.(*protocol_message.MessageSigned)
	if !ok {
		err = errors.New("TrustfulBftAdapter received a message of type MessageSigned but it is not.")
		return
	}

	// check inner type
	bftMessage, err := b.bftMessageUnmarshaller.Unmarshal(incomingMsg.GetType(), incomingMsg.GetData())
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

// PlainBftAdapter will not wrap the message using MessageTypeSigned
type PlainBftAdapter struct {
	bftMessageUnmarshaller *BftMessageUnmarshaller
}

func (p PlainBftAdapter) AdaptOgMessage(incomingMsg general_message.TransportableMessage) (msg bft.BftMessage, err error) {
	if incomingMsg.GetType() != general_message.MessageTypePlain {
		err = errors.New("PlainBftAdapter received a message of an unsupported type")
		return
	}
	iMsg := incomingMsg.(*protocol_message.MessagePlain)

	switch bft.BftMessageType(iMsg.GetType()) {
	case bft.BftMessageTypeProposal:
		fallthrough
	case bft.BftMessageTypePreVote:
		fallthrough
	case bft.BftMessageTypePreCommit:
		msg, err = p.bftMessageUnmarshaller.Unmarshal(iMsg.GetType(), iMsg.GetData())
	default:
		err = errors.New("PlainBftAdapter received a message of an unsupported inner type")
	}
	return

}

func (p PlainBftAdapter) AdaptBftMessage(outgoingMsg bft.BftMessage) (msg general_message.TransportableMessage, err error) {
	switch outgoingMsg.GetType() {
	case bft.BftMessageTypeProposal:
		omsg := outgoingMsg.(*bft.MessageProposal)
		msgBytes, err := omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		msg = protocol_message.MessagePlain{
			InnerMessageType: general_message.BinaryMessageType(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case bft.BftMessageTypePreVote:
		omsg := outgoingMsg.(*bft.MessagePreVote)
		msgBytes, err := omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		msg = protocol_message.MessagePlain{
			InnerMessageType: general_message.BinaryMessageType(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case bft.BftMessageTypePreCommit:
		omsg := outgoingMsg.(*bft.MessagePreCommit)
		msgBytes, err := omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		msg = protocol_message.MessagePlain{
			InnerMessageType: general_message.BinaryMessageType(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	default:
		err = errors.New("PlainBftAdapter received a message of an unsupported type")
	}
	return
}
