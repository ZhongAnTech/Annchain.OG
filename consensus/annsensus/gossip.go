package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

func (as *AnnSensus)gossipLoop() {
	for {
		select {
		case <-time.After(time.Second):
			if len(as.campaigns) ==0 {
				continue
			}
		  for _,c := range as.campaigns {
			  msg := &types.MessageConsensusDkgDeal{
				  Data: "hi this is a secret gossip data with rand code :" + fmt.Sprintf("%d", rand.Int63()),
				  Id:   og.MsgCounter.Get(),
			  }
              s:=  crypto.NewSigner(crypto.CryptoTypeSecp256k1)
             msg.Sinature =  s.Sign(*as.MyPrivKey,msg.SignatureTargets()).Bytes
             msg.PublikKey = as.MyPrivKey.PublicKey().Bytes
			  pk:= crypto.PublicKeyFromBytes(crypto.CryptoTypeSecp256k1,c.PublicKey)
			  as.Hub.SendToAnynomous(og.MessageTypeConsensusDkgDeal,msg,&pk)
		  }
		case <-as.close:
			log.Info("gossip loop stopped")
			return
		}

	}

}