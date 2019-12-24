package annsensus

import (
	"github.com/annchain/OG/communication"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/message"
	"github.com/annchain/OG/og/account"
)

var supportedMessageTypes = []message.GeneralMessageType{
	MessageTypeAnnsensus,
}

type AnnsensusPlugin struct {
	messageHandler   communication.GeneralMessageEventHandler
	Communicator     *ProxyAnnsensusPeerCommunicator
	AnnsensusPartner *annsensus.AnnsensusPartner
}

func (a *AnnsensusPlugin) SupportedEventHandlers() []eventbus.EventHandlerRegisterInfo {
	return []eventbus.EventHandlerRegisterInfo{}
}

func (a *AnnsensusPlugin) Start() {
	a.AnnsensusPartner.Start()
}

func (a *AnnsensusPlugin) Stop() {
	a.AnnsensusPartner.Stop()
}

func (a *AnnsensusPlugin) GetMessageEventHandler() communication.GeneralMessageEventHandler {
	return a.messageHandler
}

func (o *AnnsensusPlugin) SetOutgoing(outgoing communication.GeneralPeerCommunicatorOutgoing) {
	o.Communicator.GeneralOutgoing = outgoing
}

func NewAnnsensusPlugin(termProvider annsensus.TermIdProvider,
	myAccountProvider account.AccountProvider,
	proposalGenerator bft.ProposalGenerator,
	proposalValidator bft.ProposalValidator,
	decisionMaker bft.DecisionMaker) *AnnsensusPlugin {
	// load config first.
	config := annsensus.AnnsensusProcessorConfig{
		DisableTermChange:  false,
		DisabledConsensus:  false,
		TermChangeInterval: 60 * 1000,
		GenesisAccounts:    nil,
	}
	annsensusMessageAdapter := &DefaultAnnsensusMessageAdapter{
		unmarshaller: AnnsensusMessageUnmarshaller{},
	}

	communicator := &ProxyAnnsensusPeerCommunicator{
		AnnsensusMessageAdapter: annsensusMessageAdapter,
		GeneralOutgoing:         nil, // place for p2p peer outgoing
	}
	communicator.InitDefault()

	bftAdapter := &annsensus.PlainBftAdapter{}
	dkgAdapter := &annsensus.PlainDkgAdapter{}

	annsensusPartnerProvider := &annsensus.DefaultAnnsensusPartnerProvider{
		MyAccountProvider: myAccountProvider,
		ProposalGenerator: proposalGenerator,
		ProposalValidator: proposalValidator,
		DecisionMaker:     decisionMaker,
		BftAdatper:        bftAdapter, // for outgoing message only
		DkgAdatper:        dkgAdapter, // for outgoing message only
		PeerOutgoing:      communicator,
	}
	termHolder := annsensus.NewAnnsensusTermHolder(termProvider)

	annsensusPartner := &annsensus.AnnsensusPartner{
		Config:             config,
		BftAdapter:         bftAdapter, // for incoming message only
		DkgAdapter:         dkgAdapter, // for incoming message only
		TermProvider:       termProvider,
		TermHolder:         termHolder,
		BftPartnerProvider: annsensusPartnerProvider,
		DkgPartnerProvider: annsensusPartnerProvider,
		PeerOutgoing:       communicator,
		PeerIncoming:       communicator,
	}

	return &AnnsensusPlugin{
		messageHandler: &AnnsensusMessageMessageHandler{
			AnnsensusPartner:        annsensusPartner,
			AnnsensusMessageAdapter: annsensusMessageAdapter,
		},
		Communicator:     communicator,
		AnnsensusPartner: annsensusPartner,
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
