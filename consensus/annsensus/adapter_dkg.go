package annsensus

import (
	"errors"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/og/protocol_message"
	"github.com/annchain/OG/types/msg"
)

type DkgMessageUnmarshaller struct {
}

func (b DkgMessageUnmarshaller) Unmarshal(messageType msg.BinaryMessageType, message []byte) (outMsg dkg.DkgMessage, err error) {
	switch dkg.DkgMessageType(messageType) {
	case dkg.DkgMessageTypeDeal:
		m := &dkg.MessageDkgDeal{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case dkg.DkgMessageTypeDealResponse:
		m := &dkg.MessageDkgDealResponse{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case dkg.DkgMessageTypeGenesisPublicKey:
		m := &dkg.MessageDkgGenesisPublicKey{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case dkg.DkgMessageTypeSigSets:
		m := &dkg.MessageDkgSigSets{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	default:
		err = errors.New("message type of Dkg not supported")
	}
	return
}

// TrustfulDkgAdapter signs and validate messages using pubkey/privkey given by DKG/BLS
type TrustfulDkgAdapter struct {
	signatureProvider      account.SignatureProvider
	termProvider           TermProvider
	dkgMessageUnmarshaller *DkgMessageUnmarshaller
}

func (r *TrustfulDkgAdapter) AdaptDkgMessage(outgoingMsg dkg.DkgMessage) (msg msg.TransportableMessage, err error) {
	signed := r.Sign(outgoingMsg)
	msg = &signed
	return
}

func (r *TrustfulDkgAdapter) Sign(message dkg.DkgMessage) protocol_message.MessageSigned {
	publicKey, signature := r.signatureProvider.Sign(message.SignatureTargets())
	signed := protocol_message.MessageSigned{
		InnerMessageType: msg.BinaryMessageType(message.GetType()),
		InnerMessage:     message.SignatureTargets(),
		Signature:        signature,
		PublicKey:        publicKey,
	}
	//SessionId:     partner.CurrentTerm(),
	//PublicKey: account.PublicKey.Bytes,
	return signed
}

func NewTrustfulDkgAdapter() *TrustfulDkgAdapter {
	return &TrustfulDkgAdapter{}
}

func (b *TrustfulDkgAdapter) AdaptOgMessage(incomingMsg msg.TransportableMessage) (msg dkg.DkgMessage, err error) { // Only allows SignedOgPartnerMessage
	panic("not implemented yet")
}

type PlainDkgAdapter struct {
	dkgMessageUnmarshaller *DkgMessageUnmarshaller
}

func (p PlainDkgAdapter) AdaptOgMessage(incomingMsg msg.TransportableMessage) (msg dkg.DkgMessage, err error) {
	if incomingMsg.GetType() != protocol_message.MessageTypePlain {
		err = errors.New("PlainDkgAdapter received a message of an unsupported type")
		return
	}
	iMsg := incomingMsg.(*protocol_message.MessagePlain)

	switch dkg.DkgMessageType(iMsg.GetType()) {
	case dkg.DkgMessageTypeDeal:
		fallthrough
	case dkg.DkgMessageTypeDealResponse:
		fallthrough
	case dkg.DkgMessageTypeSigSets:
		fallthrough
	case dkg.DkgMessageTypeGenesisPublicKey:
		msg, err = p.dkgMessageUnmarshaller.Unmarshal(iMsg.GetType(), iMsg.GetData())
	default:
		err = errors.New("PlainDkgAdapter received a message of an unsupported inner type")
	}
	return
}

func (p PlainDkgAdapter) AdaptDkgMessage(outgoingMsg dkg.DkgMessage) (adaptedMessage msg.TransportableMessage, err error) {
	switch outgoingMsg.GetType() {
	case dkg.DkgMessageTypeDeal:
		omsg := outgoingMsg.(*dkg.MessageDkgDeal)
		msgBytes, err := omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = protocol_message.MessagePlain{
			InnerMessageType: msg.BinaryMessageType(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case dkg.DkgMessageTypeDealResponse:
		omsg := outgoingMsg.(*dkg.MessageDkgDealResponse)
		msgBytes, err := omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = protocol_message.MessagePlain{
			InnerMessageType: msg.BinaryMessageType(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case dkg.DkgMessageTypeGenesisPublicKey:
		omsg := outgoingMsg.(*dkg.MessageDkgGenesisPublicKey)
		msgBytes, err := omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = protocol_message.MessagePlain{
			InnerMessageType: msg.BinaryMessageType(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case dkg.DkgMessageTypeSigSets:
		omsg := outgoingMsg.(*dkg.MessageDkgSigSets)
		msgBytes, err := omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = protocol_message.MessagePlain{
			InnerMessageType: msg.BinaryMessageType(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	default:
		err = errors.New("PlainDkgAdapter received a message of an unsupported type")
	}
	return
}
