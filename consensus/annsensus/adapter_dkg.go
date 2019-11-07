package annsensus

import (
	"errors"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/types/msg"
)

type DkgMessageUnmarshaller struct {
}

func (b DkgMessageUnmarshaller) Unmarshal(messageType dkg.DkgMessageType, message []byte) (outMsg dkg.DkgMessage, err error) {
	switch messageType {
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
	termProvider           TermIdProvider
	dkgMessageUnmarshaller *DkgMessageUnmarshaller
}

func (r *TrustfulDkgAdapter) AdaptDkgMessage(outgoingMsg dkg.DkgMessage) (msg AnnsensusMessage, err error) {
	signed := r.Sign(outgoingMsg)
	msg = &signed
	return
}

func (r *TrustfulDkgAdapter) Sign(message dkg.DkgMessage) AnnsensusMessageSigned {
	publicKey, signature := r.signatureProvider.Sign(message.SignatureTargets())
	signed := AnnsensusMessageSigned{
		InnerMessageType: uint16(message.GetType()),
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

func (b *TrustfulDkgAdapter) AdaptAnnsensusMessage(incomingMsg msg.TransportableMessage) (msg dkg.DkgMessage, err error) { // Only allows SignedOgPartnerMessage
	panic("not implemented yet")
}

type PlainDkgAdapter struct {
	dkgMessageUnmarshaller *DkgMessageUnmarshaller
}

func (p PlainDkgAdapter) AdaptAnnsensusMessage(incomingMsg AnnsensusMessage) (msg dkg.DkgMessage, err error) {
	if incomingMsg.GetType() != AnnsensusMessageTypeSigned {
		err = errors.New("PlainDkgAdapter received a message of an unsupported type")
		return
	}
	iMsg := incomingMsg.(*AnnsensusMessageSigned)
	innerMessageType := dkg.DkgMessageType(iMsg.InnerMessageType)

	switch innerMessageType {
	case dkg.DkgMessageTypeDeal:
		fallthrough
	case dkg.DkgMessageTypeDealResponse:
		fallthrough
	case dkg.DkgMessageTypeSigSets:
		fallthrough
	case dkg.DkgMessageTypeGenesisPublicKey:
		msg, err = p.dkgMessageUnmarshaller.Unmarshal(innerMessageType, iMsg.InnerMessage)
	default:
		err = errors.New("PlainDkgAdapter received a message of an unsupported inner type")
	}
	return
}

func (p PlainDkgAdapter) AdaptDkgMessage(outgoingMsg dkg.DkgMessage) (adaptedMessage AnnsensusMessage, err error) {
	var msgBytes []byte
	switch outgoingMsg.GetType() {
	case dkg.DkgMessageTypeDeal:
		omsg := outgoingMsg.(*dkg.MessageDkgDeal)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = &AnnsensusMessagePlain{
			InnerMessageType: uint16(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case dkg.DkgMessageTypeDealResponse:
		omsg := outgoingMsg.(*dkg.MessageDkgDealResponse)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = &AnnsensusMessagePlain{
			InnerMessageType: uint16(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case dkg.DkgMessageTypeGenesisPublicKey:
		omsg := outgoingMsg.(*dkg.MessageDkgGenesisPublicKey)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = &AnnsensusMessagePlain{
			InnerMessageType: uint16(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case dkg.DkgMessageTypeSigSets:
		omsg := outgoingMsg.(*dkg.MessageDkgSigSets)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = &AnnsensusMessagePlain{
			InnerMessageType: uint16(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	default:
		err = errors.New("PlainDkgAdapter received a message of an unsupported type")
	}
	return
}
