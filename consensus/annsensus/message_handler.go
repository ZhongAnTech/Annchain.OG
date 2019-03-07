package annsensus

import (
	// "bytes"

	"encoding/hex"
	"github.com/annchain/OG/common/crypto"
	// "github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/dkg/pedersen"
	// "github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)



func (a *AnnSensus) HandleConsensusDkgDeal(request *types.MessageConsensusDkgDeal, peerId string) {
	log := log.WithField("me", a.id)
	if request == nil {
		log.Warn("got nil MessageConsensusDkgDeal")
		return
	}
	log.WithField("dkg data", request).WithField("from peer ", peerId).Debug("got dkg")
	pk := crypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, request.Sinature), request.SignatureTargets())
	if !ok {
		log.Warn("verify signature failed")
		return
	}
	a.dkg.gossipReqCh <- request

}

func (a *AnnSensus) HandleConsensusDkgDealResponse(request *types.MessageConsensusDkgDealResponse, peerId string) {
	log := log.WithField("me", a.id)
	if request == nil {
		log.Warn("got nil MessageConsensusDkgDealResponse")
		return
	}
	log.WithField("dkg response data", request).WithField("from peer ", peerId).Debug("got dkg response")
	pk := crypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, request.Sinature), request.SignatureTargets())
	if !ok {
		log.Warn("verify signature failed")
		return
	}
	log.Debug("response ok")

	a.dkg.gossipRespCh <- request
}

func (a *AnnSensus) HandleConsensusDkgSigSets(request *types.MessageConsensusDkgSigSets, peerId string) {
	log := log.WithField("me", a.id)
	if request == nil {
		log.Warn("got nil MessageConsensusDkgSigSets")
		return
	}
	log.WithField("data", request).WithField("from peer ", peerId).Debug("got dkg bls sigsets")
	pk := crypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, request.Sinature), request.SignatureTargets())
	if !ok {
		log.WithField("pkbls ", hex.EncodeToString(request.PkBls)).WithField("pk ", hex.EncodeToString(request.PublicKey)).WithField(
			"sig ", hex.EncodeToString(request.Sinature)).Warn("verify signature failed")
		return
	}
	log.Debug("response ok")

	a.dkg.gossipSigSetspCh <- request
}
