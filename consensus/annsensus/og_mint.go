package annsensus

import (
	"crypto"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

//ogmint is og sequencer consensus system based on OgMint consensus
type OgMint struct {
	BFTPartner *OGBFTPartner
	startProposeChan chan bool
	resetChan        chan bool
	mu               sync.RWMutex
	quit             chan bool
	ann              *AnnSensus
	c                *og.TxCreator
	JudgeNonce       func(account *account.SampleAccount) uint64
}


type   OGBFTPartner struct {
       BFTPartner
       PublicKey crypto.PublicKey
       Address   types.Address
}

func NewOgMint(nbParticipants int, Id int, sequencerTime time.Duration, judgeNonce func(me *account.SampleAccount) uint64, txcreator *og.TxCreator) *OgMint {
	p := NewBFTPartner(nbParticipants, Id, sequencerTime)
	bft:= &OGBFTPartner{
		BFTPartner:p,
	}
	om := &OgMint{
		BFTPartner:       bft,
		quit:             make(chan bool),
		startProposeChan: make(chan bool),
		resetChan:        make(chan bool),
		c:                txcreator,
	}
	om.BFTPartner.SetProposalFunc(om.ProduceProposal)
	om.JudgeNonce = judgeNonce
	return om
}

func (t *OgMint) Start() {
	go t.BFTPartner. WaiterLoop()
	go t.BFTPartner.EventLoop()
	go t.loop()
	logrus.Info("OgMint stated")
}

func (t *OgMint) Stop() {
	t.BFTPartner.Stop()
	t.quit <- true
	logrus.Info("OgMint stopped")
}


func (t *OgMint) loop() {
	log := logrus.WithField("me", t.BFTPartner.GetId())
	for {
		select {
		case <-t.quit:
			log.Info("got quit signal, OgMint loop")
		case <-t.startProposeChan:
			go t.BFTPartner.StartNewEra(0, 0)

		case msg := <-t.BFTPartner.GetOutgoingMessageChannel():
			peers := t.BFTPartner.GetPeers
			for _, peer := range peers() {
				logrus.WithFields(logrus.Fields{
					"IM":  t.BFTPartner.GetId(),
					"to":  peer.GetId(),
					"msg": msg.String(),
				}).Debug("Out")
			}
		}
	}
}

func (t *OgMint) ProduceProposal() types.Proposal {
	me := t.ann.MyAccount
	nonce := t.JudgeNonce(me)
	seq := t.c.GenerateSequencer(me.Address, t.ann.Idag.LatestSequencer().Height, nonce, &me.PrivateKey)
	if seq == nil {
		logrus.Warn("gen sequencer failed")
		panic("gen sequencer failed")
	}
	rawseq := seq.RawSequencer()
	proposal := types.SequencerProposal{
		RawSequencer: *rawseq,
	}

	return &proposal
}
