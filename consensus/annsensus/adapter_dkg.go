package annsensus

import (
	"errors"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/og/account"
)

type ProxyDkgPeerCommunicator struct {
	dkgMessageAdapter DkgMessageAdapter
	annsensusOutgoing AnnsensusPeerCommunicatorOutgoing
	pipe              chan *dkg.DkgMessageEvent
}

func NewProxyDkgPeerCommunicator(
	dkgMessageAdapter DkgMessageAdapter,
	annsensusCommunicator AnnsensusPeerCommunicatorOutgoing) *ProxyDkgPeerCommunicator {
	return &ProxyDkgPeerCommunicator{
		dkgMessageAdapter: dkgMessageAdapter,
		annsensusOutgoing: annsensusCommunicator,
		pipe:              make(chan *dkg.DkgMessageEvent),
	}
}

func (p *ProxyDkgPeerCommunicator) Broadcast(msg dkg.DkgMessage, peers []dkg.DkgPeer) {
	annsensusMessage, err := p.dkgMessageAdapter.AdaptDkgMessage(msg)
	if err != nil {
		panic("adapt should never fail")
	}
	// adapt the interface so that the request can be handled by annsensus
	annsensusPeers := make([]AnnsensusPeer, len(peers))
	for i, peer := range peers {
		adaptedValue, err := p.dkgMessageAdapter.AdaptDkgPeer(peer)
		if err != nil {
			panic("adapt should never fail")
		}
		annsensusPeers[i] = adaptedValue
	}

	p.annsensusOutgoing.Broadcast(annsensusMessage, annsensusPeers)
}

func (p *ProxyDkgPeerCommunicator) Unicast(msg dkg.DkgMessage, peer dkg.DkgPeer) {
	// adapt the interface so that the request can be handled by annsensus
	annsensusMessage, err := p.dkgMessageAdapter.AdaptDkgMessage(msg)
	if err != nil {
		panic("adapt should never fail")
	}
	annsensusPeer, err := p.dkgMessageAdapter.AdaptDkgPeer(peer)
	if err != nil {
		panic("adapt should never fail")
	}
	p.annsensusOutgoing.Unicast(annsensusMessage, annsensusPeer)
}

func (p *ProxyDkgPeerCommunicator) GetPipeOut() chan *dkg.DkgMessageEvent {
	// the channel to be consumed by the downstream.
	return p.pipe
}

func (p *ProxyDkgPeerCommunicator) GetPipeIn() chan *dkg.DkgMessageEvent {
	// the channel to be fed by other peers
	return p.pipe
}

func (p *ProxyDkgPeerCommunicator) Run() {
	// nothing to do
	return
}


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

func (r *TrustfulDkgAdapter) Sign(message dkg.DkgMessage) AnnsensusMessageDkgSigned {
	publicKey, signature := r.signatureProvider.Sign(message.SignatureTargets())
	signed := AnnsensusMessageDkgSigned{
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

func (b *TrustfulDkgAdapter) AdaptAnnsensusMessage(incomingMsg AnnsensusMessage) (msg dkg.DkgMessage, err error) { // Only allows SignedOgPartnerMessage
	panic("not implemented yet")
}

type PlainDkgAdapter struct {
	DkgMessageUnmarshaller *DkgMessageUnmarshaller
}

func (p PlainDkgAdapter) AdaptAnnsensusPeer(annPeer AnnsensusPeer) (dkg.DkgPeer, error) {
	return dkg.DkgPeer{
		Id:             annPeer.Id,
		PublicKey:      annPeer.PublicKey,
		Address:        annPeer.Address,
		PublicKeyBytes: annPeer.PublicKeyBytes,
	}, nil
}

func (p PlainDkgAdapter) AdaptDkgPeer(dkgPeer dkg.DkgPeer) (AnnsensusPeer, error) {
	return AnnsensusPeer{
		Id:             dkgPeer.Id,
		PublicKey:      dkgPeer.PublicKey,
		Address:        dkgPeer.Address,
		PublicKeyBytes: dkgPeer.PublicKeyBytes,
	}, nil
}

func (p PlainDkgAdapter) AdaptAnnsensusMessage(incomingMsg AnnsensusMessage) (msg dkg.DkgMessage, err error) {
	if incomingMsg.GetType() != AnnsensusMessageTypeDkgSigned {
		err = errors.New("PlainDkgAdapter received a message of an unsupported type")
		return
	}
	iMsg := incomingMsg.(*AnnsensusMessageDkgSigned)
	innerMessageType := dkg.DkgMessageType(iMsg.InnerMessageType)

	switch innerMessageType {
	case dkg.DkgMessageTypeDeal:
		fallthrough
	case dkg.DkgMessageTypeDealResponse:
		fallthrough
	case dkg.DkgMessageTypeSigSets:
		fallthrough
	case dkg.DkgMessageTypeGenesisPublicKey:
		msg, err = p.DkgMessageUnmarshaller.Unmarshal(innerMessageType, iMsg.InnerMessage)
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
		adaptedMessage = &AnnsensusMessageDkgPlain{
			InnerMessageType: uint16(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case dkg.DkgMessageTypeDealResponse:
		omsg := outgoingMsg.(*dkg.MessageDkgDealResponse)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = &AnnsensusMessageDkgPlain{
			InnerMessageType: uint16(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case dkg.DkgMessageTypeGenesisPublicKey:
		omsg := outgoingMsg.(*dkg.MessageDkgGenesisPublicKey)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = &AnnsensusMessageDkgPlain{
			InnerMessageType: uint16(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	case dkg.DkgMessageTypeSigSets:
		omsg := outgoingMsg.(*dkg.MessageDkgSigSets)
		msgBytes, err = omsg.MarshalMsg(nil)
		if err != nil {
			return
		}
		adaptedMessage = &AnnsensusMessageDkgPlain{
			InnerMessageType: uint16(omsg.GetType()),
			InnerMessage:     msgBytes,
		}
	default:
		err = errors.New("PlainDkgAdapter received a message of an unsupported type")
	}
	return
}
