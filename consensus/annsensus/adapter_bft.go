package annsensus

import (
	"bytes"
	"errors"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/og/protocol/ogmessage"

	"github.com/annchain/OG/types/msg"
	"github.com/sirupsen/logrus"
)

type BftMessageUnmarshaller struct {
}

func (b *BftMessageUnmarshaller) Unmarshal(messageType msg.BinaryMessageType, message []byte) (outMsg bft.BftMessage, err error) {
	switch bft.BftMessageType(messageType) {
	case bft.BftMessageTypeProposal:
		m := &bft.MessageProposal{
			Value: &bft.StringProposal{},
		}
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

func (r *TrustfulBftAdapter) AdaptBftMessage(outgoingMsg bft.BftMessage) (msg msg.TransportableMessage, err error) {
	signed := r.Sign(outgoingMsg)
	msg = &signed
	return
}

func NewTrustfulBftAdapter(
	signatureProvider account.SignatureProvider,
	termProvider TermProvider) *TrustfulBftAdapter {
	return &TrustfulBftAdapter{signatureProvider: signatureProvider, termProvider: termProvider}
}

func (r *TrustfulBftAdapter) Sign(rawMessage bft.BftMessage) ogmessage.MessageSigned {
	publicKey, signature := r.signatureProvider.Sign(rawMessage.SignatureTargets())
	signedMessage := ogmessage.MessageSigned{
		InnerMessageType: msg.BinaryMessageType(rawMessage.GetType()),
		InnerMessage:     rawMessage.SignatureTargets(),
		Signature:        signature,
		PublicKey:        publicKey,
	}
	//SessionId:     partner.CurrentTerm(),
	//PublicKey: account.PublicKey.Bytes,
	return signedMessage
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

func (b *TrustfulBftAdapter) VerifyParnterIdentity(signedMsg *ogmessage.MessageSigned) error {
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

func (b *TrustfulBftAdapter) AdaptOgMessage(incomingMsg msg.TransportableMessage) (msg bft.BftMessage, err error) { // Only allows SignedOgPartnerMessage
	if incomingMsg.GetType() != ogmessage.MessageTypeSigned {
		err = errors.New("TrustfulBftAdapter received a message of an unsupported type")
		return
	}

	signedMsg, ok := incomingMsg.(*ogmessage.MessageSigned)
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

func (p PlainBftAdapter) AdaptOgMessage(incomingMsg msg.TransportableMessage) (msg bft.BftMessage, err error) {
	if incomingMsg.GetType() != ogmessage.MessageTypePlain {
		err = errors.New("PlainBftAdapter received a message of an unsupported type")
		return
	}
	iMsg := incomingMsg.(ogmessage.MessagePlain)

	switch bft.BftMessageType(iMsg.InnerMessageType) {
	case bft.BftMessageTypeProposal:
		fallthrough
	case bft.BftMessageTypePreVote:
		fallthrough
	case bft.BftMessageTypePreCommit:
		msg, err = p.bftMessageUnmarshaller.Unmarshal(iMsg.InnerMessageType, iMsg.InnerMessage)
	default:
		err = errors.New("PlainBftAdapter received a message of an unsupported inner type")
	}
	return

}

func (p PlainBftAdapter) AdaptBftMessage(outgoingMsg bft.BftMessage) (adaptedMessage msg.TransportableMessage, err error) {
	var msgBytes []byte
	switch outgoingMsg.GetType() {
	case bft.BftMessageTypeProposal:
		omsg := outgoingMsg.(*bft.MessageProposal)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = ogmessage.MessagePlain{
			InnerMessageType: msg.BinaryMessageType(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case bft.BftMessageTypePreVote:
		omsg := outgoingMsg.(*bft.MessagePreVote)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = ogmessage.MessagePlain{
			InnerMessageType: msg.BinaryMessageType(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case bft.BftMessageTypePreCommit:
		omsg := outgoingMsg.(*bft.MessagePreCommit)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = ogmessage.MessagePlain{
			InnerMessageType: msg.BinaryMessageType(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	default:
		err = errors.New("PlainBftAdapter received a message of an unsupported type")
	}
	return
}
