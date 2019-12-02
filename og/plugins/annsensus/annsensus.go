package annsensus

import (
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/og/communication"
	"github.com/annchain/OG/og/plugins"
	"github.com/annchain/OG/types/msg"
)

var supportedMessageTypes = []msg.OgMessageType{
	MessageTypeAnnsensus,
}

type AnnsensusPlugin struct {
	messageHandler plugins.OgMessageHandler
}

func NewAnnsensusPlugin() *AnnsensusPlugin {
	// load config first.
	config := annsensus.AnnsensusProcessorConfig{
		DisableTermChange:  false,
		DisabledConsensus:  false,
		TermChangeInterval: 60 * 1000,
		GenesisAccounts:    nil,
	}
	bftAdapter := &annsensus.PlainBftAdapter{}
	dkgAdapter := &annsensus.PlainDkgAdapter{}

	communicator := &ProxyAnnsensusPeerCommunicator{
		AnnsensusMessageAdapter: nil,
		OgOutgoing:              nil,
		pipe:                    nil,
	}
	communicator.InitDefault()

	annsensusParteer := &annsensus.AnnsensusPartner{
		Config:             annsensus.AnnsensusProcessorConfig{},
		BftAdapter:         nil,
		DkgAdapter:         nil,
		TermProvider:       nil,
		TermHolder:         nil,
		BftPartnerProvider: nil,
		DkgPartnerProvider: nil,
		PeerOutgoing:       nil,
		PeerIncoming:       nil,
	}

	return &AnnsensusPlugin{
		messageHandler: &AnnsensusOgMessageHandler{
			AnnsensusPartner: annsensus.NewAnnsensusPartner(),
		},
	}
}

func (a AnnsensusPlugin) SupportedMessageTypes() []msg.OgMessageType {
	return supportedMessageTypes
}

func (a AnnsensusPlugin) GetMessageHandler() plugins.OgMessageHandler {
	return a.messageHandler
}

type AnnsensusMessageAdapter interface {
	AdaptOgMessage(incomingMsg msg.OgMessage) (annsensus.AnnsensusMessage, error)
	AdaptOgPeer(communication.OgPeer) (annsensus.AnnsensusPeer, error)

	AdaptAnnsensusMessage(outgoingMsg annsensus.AnnsensusMessage) (msg.OgMessage, error)
	AdaptAnnsensusPeer(peer annsensus.AnnsensusPeer) (communication.OgPeer, error)
}
