package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og/communicator"
	"github.com/annchain/OG/types/p2p_message"
)

// AnnsensusProcessor integrates dkg, bft and term change with vrf.
// It receives messages from
type AnnsensusProcessor struct {
	config            AnnsensusProcessorConfig
	signer            crypto.ISigner
	p2pSender         communicator.P2PSender
	myAccountProvider ConsensusAccountProvider

	quit chan bool
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
loop:
	for {
		select {
		case <-ap.quit:
			break loop
		case
		}
	}

}

func (AnnsensusProcessor) Stop() {
	panic("implement me")
}

func (AnnsensusProcessor) HandleConsensusMessage(message p2p_message.Message) {
	panic("implement me")
}
