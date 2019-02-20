package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"math/rand"
)

var CryptoType = crypto.CryptoTypeSecp256k1

func (a *AnnSensus) HandleCampaign(request *types.MessageCampaign, peerId string) {
	if request ==nil || request.Campaign ==nil {
		log.Warn("got nil MessageCampaign")
		return
	}
	cp:= request.Campaign
	//tood
	//todo send to buffer
	err := a.VrfVerify(cp.Vrf.Vrf,cp.Vrf.PublicKey, cp.Vrf.Message,cp.Vrf.Proof)
	if err!=nil {
		log.WithError(err).Debug("vrf verify failed")
		return
	}
	var partPub kyber.Point
	err = partPub.UnmarshalBinary(cp.DkgPublicKey)
	if err!=nil {
		log.WithError(err).Debug("dkg Public key  verify failed")
		return
	}
	a.campaigns[cp.Issuer] = cp
	//todo
}

func (a *AnnSensus) HandleTermChange(request *types.MessageTermChange, peerId string) {
	if request ==nil || request.TermChange ==nil {
		log.Warn("got nil MessageTermChange")
		return
	}
	//todo
}

func (a *AnnSensus) HandleConsensusDkgDeal(request *types.MessageConsensusDkgDeal, peerId string) {
	if request ==nil  {
		log.Warn("got nil MessageConsensusDkgDeal")
		return
	}
	log.WithField("dkg data",request).WithField("from peer ", peerId).Debug("got dkg")
	pk := crypto.PublicKeyFromBytes(CryptoType,request.PublikKey)
	s:= crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(CryptoType,request.Sinature),request.SignatureTargets())
	if !ok {
		log.Warn("verify signature failed")
		return
	}
	response := &types.MessageConsensusDkgDealResponse{
		Data:"this is a dkg response " + fmt.Sprintf("%d",rand.Int31()),
		Id: request.Id,
	}
	 response.Sinature = s.Sign(*a.MyPrivKey,response.SignatureTargets()).Bytes
	 response.PublikKey = a.MyPrivKey.PublicKey().Bytes
	log.WithField("response " , response).Debug("will send response")
	a.Hub.SendToAnynomous(og.MessageTypeConsensusDkgDealResponse,response,&pk)
	//todo
}

func (a *AnnSensus) HandleConsensusDkgDealResponse(request *types.MessageConsensusDkgDealResponse, peerId string) {
	if request ==nil {
		log.Warn("got nil MessageConsensusDkgDealResponse")
		return
	}
	log.WithField("dkg data",request).WithField("from peer ", peerId).Debug("got dkg response")
	pk := crypto.PublicKeyFromBytes(CryptoType,request.PublikKey)
	s:= crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(CryptoType,request.Sinature),request.SignatureTargets())
	if !ok {
		log.Warn("verify signature failed")
		return
	}
	log.Debug("response ok")
	//todo
}
