package dkg

import (
	"github.com/annchain/kyber/v3/pairing/bn256"
	"github.com/sirupsen/logrus"
	"sync"
	"testing"
	"time"
)

var TestNodes = 4

func init() {
	Formatter := new(logrus.TextFormatter)
	//Formatter.ForceColors = false
	Formatter.DisableColors = true
	Formatter.TimestampFormat = "15:04:05.000000"
	Formatter.FullTimestamp = true
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(Formatter)
	//logrus.SetReportCaller(true)

	//filenameHook := filename.NewHook()
	//filenameHook.Field = "line"
	//logrus.AddHook(filenameHook)
}

func setupPartners(termId uint32, numParts int, threshold int) (dkgPartners []*DefaultDkgPartner, partSecs []PartSec, err error) { // generate dkger
	suite := bn256.NewSuiteG2()

	dkgers, partSecs, err := SetupAllDkgers(suite, numParts, threshold)
	if err != nil {
		return
	}

	// setup partners
	peerChans := make([]chan DkgMessage, numParts)

	// prepare incoming channels
	for i := 0; i < numParts; i++ {
		peerChans[i] = make(chan DkgMessage, 5000)
	}

	dkgPartners = make([]*DefaultDkgPartner, numParts)
	partPubs := make([]PartPub, numParts)

	for i, partSec := range partSecs {
		partPubs[i] = partSec.PartPub
	}

	for i, dkger := range dkgers {
		communicator := NewDummyDkgPeerCommunicator(i, peerChans[i], peerChans)
		communicator.Run()

		partner, err := NewDefaultDkgPartner(suite, termId, numParts, threshold, partPubs, partSecs[i],
			communicator, communicator)
		if err != nil {
			panic(err)
		}
		partner.context.Dkger = dkger
		dkgPartners[i] = partner
	}
	return

}

type dummyDkgGeneratedListener struct {
	Wg       *sync.WaitGroup
	c        chan bool
	finished bool
}

func NewDummyDkgGeneratedListener(wg *sync.WaitGroup) *dummyDkgGeneratedListener {
	d := &dummyDkgGeneratedListener{
		Wg: wg,
		c:  make(chan bool),
	}
	go func() {
		for {
			<-d.c
			d.Wg.Done()
			logrus.Info("Dkg is generated")
			//break
		}
	}()
	return d
}

func (d dummyDkgGeneratedListener) GetDkgGeneratedEventChannel() chan bool {
	return d.c
}

func TestDkgPartner(t *testing.T) {
	// simulate 4 dkg partners
	termId := uint32(0)
	numParts := TestNodes
	threshold := TestNodes

	dkgPartners, partSecs, err := setupPartners(termId, numParts, threshold)
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}

	wg.Add(len(partSecs)) // no need to use partSecs, just for note
	listener := NewDummyDkgGeneratedListener(&wg)

	for _, partner := range dkgPartners {
		partner.dkgGeneratedListeners = append(partner.dkgGeneratedListeners, listener)
		partner.Start()
	}
	time.Sleep(time.Second * 5)
	for _, partner := range dkgPartners {
		partner.gossipStartCh <- true
	}

	wg.Wait()
}
