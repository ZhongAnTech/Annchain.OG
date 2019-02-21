package annsensus

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"time"
)

func (as *AnnSensus) gossipLoop() {
	for {
		select {
		case <-time.After(time.Second):
			if len(as.campaigns) == 0 {
				continue
			}

			if len(as.campaigns) < as.NbParticipants {
				log.Debug("not enough campaigns , waiting")
				continue
			}
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
				cp, ok := as.campaigns[*addr]
				if !ok {
					panic("campaign not found")
				}
				s := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
				msg.Sinature = s.Sign(*as.MyPrivKey, msg.SignatureTargets()).Bytes
				msg.PublicKey = as.MyPrivKey.PublicKey().Bytes
				pk := crypto.PublicKeyFromBytes(crypto.CryptoTypeSecp256k1, cp.PublicKey)
				as.Hub.SendToAnynomous(og.MessageTypeConsensusDkgDeal, msg, &pk)
			}
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
