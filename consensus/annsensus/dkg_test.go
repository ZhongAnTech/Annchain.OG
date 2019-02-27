package annsensus

import (
	"bytes"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"math/rand"
	"testing"
	"time"
)

type TestAnnSensus struct {
	Id int
	*AnnSensus
}


type TestMsg struct {
	message types.Message
	messageType og.MessageType
	from    int
}

func (t TestMsg)String () {
	fmt.Sprintf("from %d, type %s, msg %s",t.from,t.messageType, t.message)
}

type TestHub struct {
	Id  int
	Peers []int
	getChan func(id int ) (chan TestMsg)
	getChanByPubKey func (pub *crypto.PublicKey)(ch chan TestMsg,id int )
	OutMsg chan TestMsg
	quit chan struct{}
	As *AnnSensus
}


func ( t*TestHub)BroadcastMessage(messageType og.MessageType, message types.Message) {
	tMsg:= TestMsg{
		messageType:messageType,
		message:message,
		from:t.Id,
	}
	for _,peer:= range t.Peers{
		ch := t.getChan(peer)
		ch<-tMsg
		logrus.WithField("me ", t.Id).WithField("to peer ", peer).WithField("type ", messageType).Trace("send msg")
	}
}

func ( t*TestHub)SendToAnynomous(messageType og.MessageType, message types.Message, anyNomousPubKey *crypto.PublicKey) {
	tMsg:= TestMsg{
		messageType:messageType,
		message:message,
		from:t.Id,
	}
	ch,id := t.getChanByPubKey(anyNomousPubKey)
	ch <-tMsg
	logrus.WithField("me ", t.Id).WithField("to peer ", id).WithField("type ", messageType).Trace("send msg encr")
}

func (t *TestHub) Msgloop () {
	for {
		select {
		 case msg:= <-t.OutMsg:
		 	time.Sleep(20*time.Millisecond)
		 	logrus.WithField("my id", t.Id).WithField("msg ", msg).Debug("i got a msg")
			 switch msg.messageType {
			 case og.MessageTypeConsensusDkgDeal:
		 	request := msg.message.(*types.MessageConsensusDkgDeal)
			  t.As.HandleConsensusDkgDeal(request, fmt.Sprintf("%d",msg.from))
			 case og.MessageTypeConsensusDkgDealResponse:
				 request := msg.message.(*types.MessageConsensusDkgDealResponse)
				 t.As.HandleConsensusDkgDealResponse(request, fmt.Sprintf("%d",msg.from))
			 default:
				 logrus.WithField("my id ",t.Id ).Info("never come here , msg loop ")
				 return
		 }
		case   <-t.quit:
			logrus.WithField("mt id ",t.Id).Debug("stopped")
			return
		}
	}
}


func TestDKGMain(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	//fmt.Println(logrus.StandardLogger())
	var Anns []TestAnnSensus
	getChan :=  func (id int  ) ( ch chan TestMsg) {
		return  Anns[id].Hub.(*TestHub).OutMsg
	}
	getChanByPubKey := func (pub *crypto.PublicKey)(ch chan TestMsg,id int ) {
		for j:=0;j<4;j++ {
			if bytes.Equal( Anns[j].MyPrivKey.PublicKey().Bytes,pub.Bytes){
				return  Anns[j].Hub.(*TestHub).OutMsg,id
			}
		}
		logrus.Warn("not found for pubkey ",pub)
		return nil, 0
	}

	for j:=0;j<4;j++ {
		 as:= NewAnnSensus(crypto.CryptoTypeSecp256k1,false,4,4)
		 a:=TestAnnSensus{
		 	Id:j,
		 	AnnSensus: as,
		 }
		 var peers []int
		 for k:=0;k<4;k++ {
		 	if k==j {
		 		continue
			}
		 	peers = append(peers,k )
		 }
		  _ ,priv,_ := crypto.NewSigner(crypto.CryptoTypeSecp256k1).RandomKeyPair()
		 as.MyPrivKey = &priv
		 as.Idag = &DummyDag{}
		 hub:= TestHub{
		 	Id:j,
		 	Peers: peers,
		 	OutMsg:make(chan TestMsg),
		 	getChan:getChan,
		 	getChanByPubKey:getChanByPubKey,
		 	quit:make(chan struct{}),
		 	As:as,
		 }
		 a.Hub = &hub
		 a.AnnSensus = as
		 Anns = append(Anns,a)
	}

	//start here
	//collect campaigns
	og.MsgCountInit()
	var cps types.Txis
	for _ ,ann := range Anns {
		cps = append(cps,  ann.GenCampaign())
	}
	for _ , ann:= range Anns{
		ann.Start()
	}
	time.Sleep(50*time.Millisecond)
	for _ , ann:= range Anns{
		ann.ConsensusTXConfirmed <-cps
	}

	time.Sleep(time.Second*10)
	for _ , ann:= range Anns{
		ann.Stop()
	}

}


func (as *TestAnnSensus)Start () {
	as.campaignFlag = true
	logrus.Info("AnnSensus Start")
	logrus.Tracef("campaignFlag: %v", as.campaignFlag)
	go as.gossipLoop()

	go as.loop()
	go as.Hub.(*TestHub).Msgloop()
	logrus.Info("started ann  ", as.Id)
}


func (a *TestAnnSensus)Stop () {
    a.AnnSensus.Stop()
    a.Hub.(*TestHub).quit <- struct{}{}
    logrus.Info("stopped ann  ", a.Id)
}


func (as *TestAnnSensus)GenCampaign() ( *types.Campaign) {
	pubKey := as.GenerateDkg()
	// generate campaign.
	camp := as.genCamp(pubKey)
	as.newCampaign(camp)
	ok:= as.VerifyCampaign(camp)
	if !ok {
		panic(ok)
	}
	return camp
}


func (as *TestAnnSensus)newCampaign(cp* types.Campaign) {
	cp.GetBase().PublicKey = as.MyPrivKey.PublicKey().Bytes
	cp.GetBase().AccountNonce = rand.Uint64()
	cp.Issuer = as.MyPrivKey.PublicKey().Address()
	s := crypto.NewSigner(as.cryptoType)
	cp.GetBase().Signature = s.Sign(*as.MyPrivKey, cp.SignatureTargets()).Bytes
	cp.GetBase().Weight =uint64(as.Id)
	cp.Height = uint64(as.Id)
	cp.GetBase().Hash =  cp.CalcTxHash()
	return
}