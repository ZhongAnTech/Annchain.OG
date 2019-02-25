package annsensus

import (
	"bytes"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/dkg/pedersen"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

func (as *AnnSensus) gossipLoop() {
	for {
		// TODO case dealing dkg
		select {
		case <-as.termChgSignal:

			as.partner.GenerateDKGer()
			deals, err := as.partner.Dkger.Deals()
			if err != nil {
				log.WithError(err).Error("generate dkg deal error")
			}
			for i, deal := range deals {
				data, _ := deal.MarshalMsg(nil)
				msg := &types.MessageConsensusDkgDeal{
					Data: data,
					Id:   og.MsgCounter.Get(),
				}
				addr := as.GetPartnerAddressByIndex(i)
				if addr == nil {
					panic("address not found")
				}
				cp, ok := as.candidates[*addr]
				if !ok {
					panic("campaign not found")
				}
				s := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
				msg.Sinature = s.Sign(*as.MyPrivKey, msg.SignatureTargets()).Bytes
				msg.PublicKey = as.MyPrivKey.PublicKey().Bytes
				pk := crypto.PublicKeyFromBytes(crypto.CryptoTypeSecp256k1, cp.PublicKey)
				as.Hub.SendToAnynomous(og.MessageTypeConsensusDkgDeal, msg, &pk)
			}

		case request := <-as.dkgReqCh:
			var deal dkg.Deal
			_, err := deal.UnmarshalMsg(request.Data)
			if err != nil {
				log.Warn("unmarshal failed failed")
			}
			if !as.campaignFlag {
				//not a consensus partner
				log.Warn("why send to me")
				return
			}
			var cp *types.Campaign
			for _, v := range as.candidates {
				if bytes.Equal(v.PublicKey, request.PublicKey) {
					cp = v
					break
				}
			}
			if cp == nil {
				log.WithField("deal ", request).Warn("not found  dkg  partner for deal")
				return
			}
			_, ok := as.partner.adressIndex[cp.Issuer]
			if !ok {
				log.WithField("deal ", request).Warn("not found  dkg  partner for deal")
				return
			}
			responseDeal, err := as.partner.Dkger.ProcessDeal(&deal)
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
			signer := crypto.NewSigner(as.cryptoType)
			response.Sinature = signer.Sign(*as.MyPrivKey, response.SignatureTargets()).Bytes
			response.PublicKey = as.MyPrivKey.PublicKey().Bytes
			log.WithField("response ", response).Debug("will send response")
			//broadcast response to all partner
			as.Hub.BroadcastMessage(og.MessageTypeConsensusDkgDealResponse, response)

		case response := <-as.dkgRespCh:
			var resp dkg.Response
			_, err := resp.UnmarshalMsg(response.Data)
			if err != nil {
				log.WithError(err).Warn("verify signature failed")
				return
			}
			//broadcast  continue
			as.Hub.BroadcastMessage(og.MessageTypeConsensusDkgDealResponse, response)
			if !as.campaignFlag {
				//not a consensus partner
				return
			}
			var cp *types.Campaign
			for _, v := range as.candidates {
				if bytes.Equal(v.PublicKey, response.PublicKey) {
					cp = v
					break
				}
			}
			if cp == nil {
				log.WithField("deal ", response).Warn("not found  dkg  partner for deal")
				return
			}
			_, ok := as.partner.adressIndex[cp.Issuer]
			if !ok {
				log.WithField("deal ", response).Warn("not found  dkg  partner for deal")
				return
			}
			just, err := as.partner.Dkger.ProcessResponse(&resp)
			if err != nil {
				log.WithError(err).Warn("ProcessResponse failed")
				return
			}
			as.partner.responseNumber++
			if as.partner.responseNumber > (as.partner.NbParticipants-1)*(as.partner.NbParticipants-1) {
				log.Info("got response done")
				jointPub, err := as.partner.RecoverPub()
				if err != nil {
					log.WithError(err).Warn("get recover pub key fail")
				}
				log.WithField("bls key ", jointPub).Info("joint pubkey ")

			}
			log.WithField("response number", as.partner.responseNumber).Trace("dkg")
			_ = just

		case <-as.close:
			log.Info("gossip loop stopped")
			return
		}
	}
}

func (as *AnnSensus) GetPartnerAddressByIndex(i int) *types.Address {

	for k, v := range as.partner.adressIndex {
		if v == i {
			return &k
		}
	}
	return nil
}
