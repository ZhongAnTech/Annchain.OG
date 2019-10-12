// general_message is the package for low level p2p communications
// the content in BinaryMessage should either be sent directly to the others,
// or be wrapped by another BinaryMessage.
// all high level messages should have a way to convert itself to the low level format

package general_message

type BinaryMessageType uint16

// BinaryMessage stores data that can be directly sent to the others, or be wrapped by another BinaryMessage.
type BinaryMessage struct {
	Type BinaryMessageType
	Data []byte
}

// TransportableMessage is the message that can be convert to BinaryMessage
type TransportableMessage interface {
	GetType() BinaryMessageType
	GetData() []byte
	ToBinary() BinaryMessage
	FromBinary([]byte) error
	String() string
}

// og protocol message codes
// TODO: use MessageTypeManager to manage global messages
// basic messages ids range from [0, 100)
// bft consensus: [100, 200)
// dkg: [200, 300)
const (
	// Protocol messages belonging to OG/01
	StatusMsg BinaryMessageType = iota + 0
	MessageTypePing
	MessageTypePong
	MessageTypeSyncRequest
	MessageTypeSyncResponse
	MessageTypeFetchByHashRequest
	MessageTypeFetchByHashResponse
	MessageTypeNewTx
	MessageTypeNewSequencer
	MessageTypeNewTxs
	MessageTypeSequencerHeader

	MessageTypeBodiesRequest
	MessageTypeBodiesResponse

	MessageTypeTxsRequest
	MessageTypeTxsResponse
	MessageTypeHeaderRequest
	MessageTypeHeaderResponse

	//for optimizing network
	MessageTypeGetMsg
	MessageTypeDuplicate
	MessageTypeControl

	//for consensus
	MessageTypeCampaign
	MessageTypeTermChange

	MessageTypeArchive
	MessageTypeActionTX

	//move to dkg package
	//MessageTypeConsensusDkgDeal
	//MessageTypeConsensusDkgDealResponse
	//MessageTypeConsensusDkgSigSets
	//MessageTypeConsensusDkgGenesisPublicKey

	MessageTypeTermChangeRequest
	MessageTypeTermChangeResponse

	MessageTypeSecret //encrypted message

	// move to bft package
	//MessageTypeProposal
	//MessageTypePreVote
	//MessageTypePreCommit

	MessageTypeOg01Length //og01 length

	// Protocol messages belonging to og/02

	GetNodeDataMsg
	NodeDataMsg
	GetReceiptsMsg
	MessageTypeOg02Length

	MessageTypeNewArchive
	MessageTypeNewActionTx
)
