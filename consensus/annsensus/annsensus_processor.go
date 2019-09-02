package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/og/communicator"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/sirupsen/logrus"
	"sync"
)

// AnnsensusProcessor integrates dkg, bft and term change with vrf.
// It receives messages from
type AnnsensusProcessor struct {
	incomingChannel   chan p2p_message.Message
	config            AnnsensusProcessorConfig
	p2pSender         communicator.P2PSender
	myAccountProvider account.AccountProvider

	BftOperator bft.BftOperator
	DkgOperator dkg.DkgOperator

	quit   chan bool
	quitWg sync.WaitGroup
}

type AnnsensusProcessorConfig struct {
	DisableTermChange  bool
	DisabledConsensus  bool
	TermChangeInterval int
	GenesisAccounts    crypto.PublicKeys
	PartnerNum         int
}

func NewAnnsensusProcessor(config AnnsensusProcessorConfig) *AnnsensusProcessor {
	if config.DisabledConsensus {
		config.DisableTermChange = true
	}
	if !config.DisabledConsensus {
		if config.TermChangeInterval <= 0 && !config.DisableTermChange {
			panic("require termChangeInterval")
		}
		if len(config.GenesisAccounts) < config.PartnerNum && !config.DisableTermChange {
			panic("need more account")
		}
		if config.PartnerNum < 2 {
			panic(fmt.Sprintf("BFT needs at least 2 nodes, currently %d", config.PartnerNum))
		}
	}

	return &AnnsensusProcessor{
		config: config,
		quit:   make(chan bool),
	}
}

func (ap *AnnsensusProcessor) Start() {
	log.Info("AnnSensus Start")
	if ap.config.DisabledConsensus {
		log.Warn("annsensus disabled")
		return
	}
	ap.quitWg.Add(1)

loop:
	for {
		select {
		case <-ap.quit:
			ap.BftOperator.Stop()
			ap.DkgOperator.Stop()
			ap.quitWg.Done()
			break loop
		case msg := <-ap.incomingChannel:
			ap.HandleConsensusMessage(msg)
		}
	}

}

func (ap *AnnsensusProcessor) Stop() {
	ap.quitWg.Wait()
	logrus.Debug("AnnsensusProcessor stopped")
}

func (AnnsensusProcessor) HandleConsensusMessage(message p2p_message.Message) {
	switch message.(type) {
	case bft.BftMessage:

	}
}
