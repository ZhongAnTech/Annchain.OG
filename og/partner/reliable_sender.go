package partner

import (
	"bytes"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/consensus/annsensus/bft"
	"github.com/sirupsen/logrus"
)

// ConsensusAccountProvider provides public key and private key for signing consensus messages
type ConsensusAccountProvider interface {
	PublicKey() crypto.PublicKey
	PrivateKey() crypto.PrivateKey
}

// TrustfulPartnerCommunicator signs and validate messages using pubkey/privkey given by DKG/BLS
// It provides Trustful communication between partners with pubkeys
// All messages received from TrustfulPartnerCommunicator is considered crypto safe and sender verified.
type TrustfulPartnerCommunicator struct {
	incomingChannel   chan bft.BftMessage
	Signer            crypto.ISigner
	TermProvider      DkgTermProvider //TODO：not its job.
	MyAccountProvider ConsensusAccountProvider
}

func NewTrustfulPeerCommunicator(signer crypto.ISigner, termProvider DkgTermProvider,
	myAccountProvider ConsensusAccountProvider) *TrustfulPartnerCommunicator {
	return &TrustfulPartnerCommunicator{
		incomingChannel:   make(chan bft.BftMessage, 20),
		Signer:            signer,
		TermProvider:      termProvider,
		MyAccountProvider: myAccountProvider,
	}
}

// SignedOgParnterMessage is the message that is signed by partner.
// Consensus layer does not need to care about the signing. It is TrustfulPartnerCommunicator's job
type SignedOgParnterMessage struct {
	bft.BftMessage
	TermId     uint32
	ValidRound int
	//PublicKey  []byte
	Signature hexutil.Bytes
	PublicKey hexutil.Bytes
}

func (r *TrustfulPartnerCommunicator) Sign(msg bft.BftMessage) SignedOgParnterMessage {
	signed := SignedOgParnterMessage{
		BftMessage: msg,
		Signature:  r.Signer.Sign(r.MyAccountProvider.PrivateKey(), msg.Payload.SignatureTargets()).Bytes,
		TermId:     r.TermProvider.CurrentDkgTerm(),
		PublicKey:  r.MyAccountProvider.PublicKey().Bytes,
	}
	return signed
	//
	//switch msg.Type {
	//case bft.BftMessageTypeProposal:
	//
	//case bft.BftMessageTypePreVote:
	//	prevote := msg.Payload.(*bft.MessagePreVote)
	//	signed := SignedOgParnterMessage{
	//		BftMessage: msg,
	//		Signature:  r.Signer.Sign(r.MyAccountProvider.PrivateKey(), prevote.SignatureTargets()).Bytes,
	//		TermId:     r.TermProvider.CurrentDkgTerm(),
	//		PublicKey:  r.MyAccountProvider.PublicKey().Bytes,
	//	}
	//	return signed
	//case bft.BftMessageTypePreCommit:
	//	preCommit := msg.Payload.(bft.MessagePreCommit)
	//	signed := SignedOgParnterMessage{
	//		BftMessage: msg,
	//		Signature:  r.Signer.Sign(r.MyAccountProvider.PrivateKey(), preCommit.SignatureTargets()).Bytes,
	//		TermId:     r.TermProvider.CurrentDkgTerm(),
	//		PublicKey:  r.MyAccountProvider.PublicKey().Bytes,
	//	}
	//
	//	// TODO: dkg sign
	//	//if preCommit.Idv != nil {
	//	//	sig, err := b.dkg.Sign(preCommit.Idv.ToBytes(), b.DKGTermId)
	//	//	if err != nil {
	//	//		logev.WithError(err).Error("sign error")
	//	//		panic(err)
	//	//	}
	//	//	preCommit.BlsSignature = sig
	//	//}
	//	return signed
	//default:
	//	panic("never come here unknown type")
	//}
}

func (r *TrustfulPartnerCommunicator) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
	// signed := r.Sign(msg)
	// TODO: send using p2p
}

func (r *TrustfulPartnerCommunicator) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	// signed := r.Sign(msg)
	// TODO: send using p2p
}

// GetIncomingChannel provides a channel for
func (r *TrustfulPartnerCommunicator) GetIncomingChannel() chan bft.BftMessage {
	return r.incomingChannel
}

func (b *TrustfulPartnerCommunicator) VerifyParnterIdentity(publicKey crypto.PublicKey, sourcePartner int) bool {
	peers := b.BFTPartner.GetPeers()
	if sourcePartner < 0 || sourcePartner > len(peers)-1 {
		logrus.WithField("len partner ", len(peers)).WithField("sr ", sourcePartner).Warn("sourceId error")
		return false
	}
	partner := peers[sourcePartner].(*OGBFTPartner)
	if bytes.Equal(partner.PublicKey.Bytes, publicKey.Bytes) {
		return true
	}
	logrus.Trace(publicKey.String(), " ", partner.PublicKey.String())
	return false

}
