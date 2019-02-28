package annsensus

import (
	"bytes"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/sha3"
	"github.com/annchain/OG/common/gcache"
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
	message     types.Message
	messageType og.MessageType
	from        int
}

func (t *TestMsg) GetHash() types.Hash {
	data, _ := t.message.MarshalMsg(nil)
	h := sha3.New256()
	h.Write(data)
	b := h.Sum(nil)
	hash := types.Hash{}
	hash.MustSetBytes(b, types.PaddingNone)
	return hash
}

func (t TestMsg) String() {
	fmt.Sprintf("from %d, type %s, msg %s", t.from, t.messageType, t.message)
}

type TestHub struct {
	Id              int
	Peers           []int
	sendMsgToChan   sendMsgToChanFunc
	sendMsgByPubKey sendMsgByPubKeyFunc
	OutMsg          chan TestMsg
	quit            chan struct{}
	As              *AnnSensus
	msgCache        gcache.Cache
}

type sendMsgToChanFunc func(id int, mdg TestMsg)
type sendMsgByPubKeyFunc func(pub *crypto.PublicKey, msg TestMsg)

func NewtestHub(id int, peers []int, sendMsgToChan sendMsgToChanFunc, sendMsgByPubKey sendMsgByPubKeyFunc, as *AnnSensus) *TestHub {
	return &TestHub{
		Id:              id,
		Peers:           peers,
		OutMsg:          make(chan TestMsg),
		sendMsgToChan:   sendMsgToChan,
		sendMsgByPubKey: sendMsgByPubKey,
		quit:            make(chan struct{}),
		As:              as,
		msgCache:        gcache.New(1000).Expiration(time.Minute).Simple().Build(),
	}
}

func (t *TestHub) BroadcastMessage(messageType og.MessageType, message types.Message) {
	tMsg := TestMsg{
		messageType: messageType,
		message:     message,
		from:        t.Id,
	}
	for _, peer := range t.Peers {
		t.sendMsgToChan(peer, tMsg)
		logrus.WithField("me ", t.Id).WithField("to peer ", peer).WithField("type ", messageType).Trace("send msg")
	}
}

func (t *TestHub) SendToAnynomous(messageType og.MessageType, message types.Message, anyNomousPubKey *crypto.PublicKey) {
	tMsg := TestMsg{
		messageType: messageType,
		message:     message,
		from:        t.Id,
	}
	t.sendMsgByPubKey(anyNomousPubKey, tMsg)

}

func (t *TestHub) loop() {
	for {
		select {
		case msg := <-t.OutMsg:
			time.Sleep(10 * time.Millisecond)
			hash := msg.GetHash()
			logrus.WithField("my id", t.Id).WithField("msg ", msg).Debug("i got a msg")
			switch msg.messageType {
			case og.MessageTypeConsensusDkgDeal:
				request := msg.message.(*types.MessageConsensusDkgDeal)
				if _, err := t.msgCache.GetIFPresent(hash); err == nil {
					logrus.WithField("me  ", t.Id).WithField("from ", msg.from).WithField("msg type",
						msg.messageType).WithField("msg ", len(request.Data)).Warn("duplicate dkg msg")
					continue
				}
				go t.As.HandleConsensusDkgDeal(request, fmt.Sprintf("%d", msg.from))
			case og.MessageTypeConsensusDkgDealResponse:
				request := msg.message.(*types.MessageConsensusDkgDealResponse)
				if _, err := t.msgCache.GetIFPresent(hash); err == nil {
					logrus.WithField("me  ", t.Id).WithField("from ", msg.from).WithField("msg type",
						msg.messageType).WithField("msg ", len(request.Data)).Warn("duplicate response  msg")
					continue
				}
				go t.As.HandleConsensusDkgDealResponse(request, fmt.Sprintf("%d", msg.from))
			default:
				logrus.WithField("my id ", t.Id).Info("never come here , msg loop ")
				return
			}
			t.msgCache.Set(hash, struct{}{})
		case <-t.quit:
			logrus.WithField("mt id ", t.Id).Debug("stopped")
			return
		}
	}
}

func TestDKGMain(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	//fmt.Println(logrus.StandardLogger())
	var Anns []TestAnnSensus
	sendMsgToChan := func(id int, msg TestMsg) {
		Anns[id].Hub.(*TestHub).OutMsg <- msg
		var resp *types.MessageConsensusDkgDealResponse
		if msg.messageType == og.MessageTypeConsensusDkgDealResponse {
			resp = msg.message.(*types.MessageConsensusDkgDealResponse)
		}
		logrus.WithField("me ", msg.from).WithField("to peer ", id).WithField("type ",
			msg.messageType).WithField("msg ", resp).Trace("send msg")
		return
	}
	sendMsgByPubKey := func(pub *crypto.PublicKey, msg TestMsg) {
		for j := 0; j < 4; j++ {
			if bytes.Equal(Anns[j].MyPrivKey.PublicKey().Bytes, pub.Bytes) {
				Anns[j].Hub.(*TestHub).OutMsg <- msg
				logrus.WithField("from peer", msg.from).WithField("to peer ", j).WithField("type ",
					msg.messageType).Trace("send msg encr")
				return
			}
		}
		logrus.Warn("not found for pubkey ", pub)
		return
	}

	for j := 0; j < 4; j++ {
		as := NewAnnSensus(crypto.CryptoTypeSecp256k1, true, 4, 4)
		a := TestAnnSensus{
			Id:        j,
			AnnSensus: as,
		}
		var peers []int
		for k := 0; k < 4; k++ {
			if k == j {
				continue
			}
			peers = append(peers, k)
		}
		_, priv, _ := crypto.NewSigner(crypto.CryptoTypeSecp256k1).RandomKeyPair()
		as.MyPrivKey = &priv
		as.Idag = &DummyDag{}
		a.Hub = NewtestHub(j, peers, sendMsgToChan, sendMsgByPubKey, as)
		a.AnnSensus = as
		logrus.WithField("addr ", a.MyPrivKey.PublicKey().Address().TerminalString()).Debug("gen hub ", a.Hub)
		Anns = append(Anns, a)
	}

	//start here
	//collect campaigns
	og.MsgCountInit()
	var cps types.Txis
	for _, ann := range Anns {
		cps = append(cps, ann.GenCampaign())
	}
	for _, ann := range Anns {
		ann.Start()
	}
	time.Sleep(20 * time.Millisecond)
	for _, ann := range Anns {
		ann.ConsensusTXConfirmed <- cps
	}

	time.Sleep(time.Second * 10)
	for _, ann := range Anns {
		ann.Stop()
	}

}

func (as *TestAnnSensus) Start() {
	as.campaignFlag = false
	logrus.Info("AnnSensus Start")
	logrus.Tracef("campaignFlag: %v", as.campaignFlag)

	go as.AnnSensus.Start()
	go as.Hub.(*TestHub).loop()
	logrus.Info("started ann  ", as.Id)
}

func (a *TestAnnSensus) Stop() {
	a.AnnSensus.Stop()
	a.Hub.(*TestHub).quit <- struct{}{}
	logrus.Info("stopped ann  ", a.Id)
}

func (as *TestAnnSensus) GenCampaign() *types.Campaign {
	// generate campaign.
	camp := as.genCamp(as.dkg.pk)
	as.newCampaign(camp)
	ok := as.VerifyCampaign(camp)
	if !ok {
		panic(ok)
	}
	return camp
}

func (as *TestAnnSensus) newCampaign(cp *types.Campaign) {
	cp.GetBase().PublicKey = as.MyPrivKey.PublicKey().Bytes
	cp.GetBase().AccountNonce = rand.Uint64()
	cp.Issuer = as.MyPrivKey.PublicKey().Address()
	s := crypto.NewSigner(as.cryptoType)
	cp.GetBase().Signature = s.Sign(*as.MyPrivKey, cp.SignatureTargets()).Bytes
	cp.GetBase().Weight = uint64(as.Id)
	cp.Height = uint64(as.Id)
	cp.GetBase().Hash = cp.CalcTxHash()
	return
}
