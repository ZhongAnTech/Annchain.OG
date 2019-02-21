package annsensus

import (
	"bytes"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/dkg/pedersen"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

var CryptoType = crypto.CryptoTypeSecp256k1

func (a *AnnSensus) HandleCampaign(request *types.MessageCampaign, peerId string) {
	if request == nil || request.Campaign == nil {
		log.Warn("got nil MessageCampaign")
		return
	}
	cp := request.Campaign
	//tood
	//todo send to buffer
	err := a.VrfVerify(cp.Vrf.Vrf, cp.Vrf.PublicKey, cp.Vrf.Message, cp.Vrf.Proof)
	if err != nil {
		log.WithError(err).Debug("vrf verify failed")
		return
	}
	var partPub kyber.Point
	err = partPub.UnmarshalBinary(cp.DkgPublicKey)
	if err != nil {
		log.WithError(err).Debug("dkg Public key  verify failed")
		return
	}
	if _, ok := a.campaigns[cp.Issuer]; !ok {
		log.WithField("campaign", cp).Debug("duplicate campaign ")
	}
	a.partner.PartPubs = append(a.partner.PartPubs, partPub)
	a.campaigns[cp.Issuer] = cp
	a.partner.adressIndex[cp.Issuer] = len(a.partner.PartPubs) - 1
	//todo
}

func (a *AnnSensus) HandleTermChange(request *types.MessageTermChange, peerId string) {
	if request == nil || request.TermChange == nil {
		log.Warn("got nil MessageTermChange")
		return
	}
	//todo
}

func (a *AnnSensus) HandleConsensusDkgDeal(request *types.MessageConsensusDkgDeal, peerId string) {
	if request == nil {
		log.Warn("got nil MessageConsensusDkgDeal")
		return
	}
	log.WithField("dkg data", request).WithField("from peer ", peerId).Debug("got dkg")
	pk := crypto.PublicKeyFromBytes(CryptoType, request.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(CryptoType, request.Sinature), request.SignatureTargets())
	if !ok {
		log.Warn("verify signature failed")
		return
	}
	var deal dkg.Deal
	_, err := deal.UnmarshalMsg(request.Data)
	if err != nil {
		log.Warn("unmarshal failed failed")
	}
	if !a.doCamp {
		//not a consensus partner
		log.Warn("why send to me")
		return
	}
	var cp *types.Campaign
	for _, v := range a.campaigns {
		if bytes.Equal(v.PublicKey, request.PublicKey) {
			cp = v
			break
		}
	}
	if cp == nil {
		log.WithField("deal ", request).Warn("not found  dkg  partner for deal")
		return
	}
	_, ok = a.partner.adressIndex[cp.Issuer]
	if !ok {
		log.WithField("deal ", request).Warn("not found  dkg  partner for deal")
		return
	}
	responseDeal, err := a.partner.Dkger.ProcessDeal(&deal)
	if err != nil {
		log.WithField("deal ", request).WithError(err).Warn("  partner process error")
		return
	}
	respData, err := responseDeal.MarshalMsg(nil)
	if err != nil {
		log.WithField("deal ", request).WithError(err).Warn("  partner process error")
		return
	}
	response := &types.MessageConsensusDkgDealResponse{
		Data: respData,
		Id:   request.Id,
	}
	response.Sinature = s.Sign(*a.MyPrivKey, response.SignatureTargets()).Bytes
	response.PublicKey = a.MyPrivKey.PublicKey().Bytes
	log.WithField("response ", response).Debug("will send response")
	//broadcast response to all partner
	a.Hub.BroadcastMessage(og.MessageTypeConsensusDkgDealResponse, response)
	//todo
}

func (a *AnnSensus) HandleConsensusDkgDealResponse(request *types.MessageConsensusDkgDealResponse, peerId string) {
	if request == nil {
		log.Warn("got nil MessageConsensusDkgDealResponse")
		return
	}
	log.WithField("dkg data", request).WithField("from peer ", peerId).Debug("got dkg response")
	pk := crypto.PublicKeyFromBytes(CryptoType, request.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(CryptoType, request.Sinature), request.SignatureTargets())
	if !ok {
		log.Warn("verify signature failed")
		return
	}
	log.Debug("response ok")
	var resp dkg.Response
	_, err := resp.UnmarshalMsg(request.Data)
	if err != nil {
		log.WithError(err).Warn("verify signature failed")
		return
	}
	//broadcast  continue
	a.Hub.BroadcastMessage(og.MessageTypeConsensusDkgDealResponse, request)
	if !a.doCamp {
		//not a consensus partner
		return
	}
	var cp *types.Campaign
	for _, v := range a.campaigns {
		if bytes.Equal(v.PublicKey, request.PublicKey) {
			cp = v
			break
		}
	}
	if cp == nil {
		log.WithField("deal ", request).Warn("not found  dkg  partner for deal")
		return
	}
	_, ok = a.partner.adressIndex[cp.Issuer]
	if !ok {
		log.WithField("deal ", request).Warn("not found  dkg  partner for deal")
		return
	}
	just, err := a.partner.Dkger.ProcessResponse(&resp)
	if err != nil {
		log.WithError(err).Warn("ProcessResponse failed")
		return
	}
	a.partner.responseNumber++
	if a.partner.responseNumber > (a.partner.NbParticipants-1)*(a.partner.NbParticipants-1) {
		log.Info("got response done")
		_, err = a.partner.RecoverPub()
		if err != nil {
			log.WithError(err).Warn("get recover pub key fail")
		}
	}
	log.WithField("response number", a.partner.responseNumber).Trace("dkg")
	_ = just
	//todo
}
