package annsensus

import (
	"github.com/annchain/OG/communication"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/message"
)

var supportedMessageTypes = []message.GeneralMessageType{
	MessageTypeAnnsensus,
}

type AnnsensusPlugin struct {
	messageHandler communication.GeneralMessageEventHandler
}

func (a *AnnsensusPlugin) GetMessageEventHandler() communication.GeneralMessageEventHandler {
	return a.messageHandler
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
		GeneralOutgoing:         nil,
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
		messageHandler: &AnnsensusMessageMessageHandler{
			AnnsensusPartner: annsensus.NewAnnsensusPartner(),
		},
	}
}

func (a AnnsensusPlugin) SupportedMessageTypes() []message.GeneralMessageType {
	return supportedMessageTypes
}

type AnnsensusMessageAdapter interface {
	AdaptGeneralMessage(incomingMsg message.GeneralMessage) (annMessage annsensus.AnnsensusMessage, err error)
	AdaptGeneralPeer(gnrPeer message.GeneralPeer) (annsensus.AnnsensusPeer, error)

	AdaptAnnsensusMessage(outgoingMsg annsensus.AnnsensusMessage) (msg message.GeneralMessage, err error)
	AdaptAnnsensusPeer(annPeer annsensus.AnnsensusPeer) (message.GeneralPeer, error)
}
