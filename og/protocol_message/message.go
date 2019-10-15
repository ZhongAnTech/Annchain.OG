// general_message is the package for low level p2p communications
// the content in BinaryMessage should either be sent directly to the others,
// or be wrapped by another BinaryMessage.
// all high level messages should have a way to convert itself to the low level format

package protocol_message

import (
	"github.com/annchain/OG/types/msg"
)

// og protocol message codes
// TODO: use MessageTypeManager to manage global messages
// basic messages ids range from [0, 100)
// bft consensus: [100, 200)
// dkg: [200, 300)
// campaign: [300, 400)
const (
	// Protocol messages belonging to OG/01
	StatusMsg msg.BinaryMessageType = iota + 0
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

	//move to campaign
	//MessageTypeCampaign
	//MessageTypeTermChange

	MessageTypeArchive
	MessageTypeActionTX

	//move to dkg package
	//MessageTypeConsensusDkgDeal
	//MessageTypeConsensusDkgDealResponse
	//MessageTypeConsensusDkgSigSets
	//MessageTypeConsensusDkgGenesisPublicKey

	//move to campaign
	//MessageTypeTermChangeRequest
	//MessageTypeTermChangeResponse

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
	MessageTypeEncrypted
	MessageTypeSigned
	MessageTypePlain
)

//func (mt msg.BinaryMessageType) String() string {
//	switch mt {
//	case StatusMsg:
//		return "StatusMsg"
//	case MessageTypePing:
//		return "MessageTypePing"
//	case MessageTypePong:
//		return "MessageTypePong"
//	case MessageTypeFetchByHashRequest:
//		return "MessageTypeFetchByHashRequest"
//	case MessageTypeFetchByHashResponse:
//		return "MessageTypeFetchByHashResponse"
//	case MessageTypeNewTx:
//		return "MessageTypeNewTx"
//	case MessageTypeNewSequencer:
//		return "MessageTypeNewSequencer"
//	case MessageTypeNewTxs:
//		return "MessageTypeNewTxs"
//	case MessageTypeSequencerHeader:
//		return "MessageTypeSequencerHeader"
//
//	case MessageTypeBodiesRequest:
//		return "MessageTypeBodiesRequest"
//	case MessageTypeBodiesResponse:
//		return "MessageTypeBodiesResponse"
//	case MessageTypeTxsRequest:
//		return "MessageTypeTxsRequest"
//	case MessageTypeTxsResponse:
//		return "MessageTypeTxsResponse"
//	case MessageTypeHeaderRequest:
//		return "MessageTypeHeaderRequest"
//	case MessageTypeHeaderResponse:
//		return "MessageTypeHeaderResponse"
//
//		//for optimizing network
//	case MessageTypeGetMsg:
//		return "MessageTypeGetMsg"
//	case MessageTypeDuplicate:
//		return "MessageTypeDuplicate"
//	case MessageTypeControl:
//		return "MessageTypeControl"
//
//		//for consensus
//	case MessageTypeCampaign:
//		return "MessageTypeCampaign"
//	case MessageTypeTermChange:
//		return "MessageTypeTermChange"
//	case MessageTypeArchive:
//		return "MessageTypeArchive"
//	case MessageTypeActionTX:
//		return "MessageTypeActionTX"
//
//	//case MessageTypeConsensusDkgDeal:
//	//	return "MessageTypeConsensusDkgDeal"
//	//case MessageTypeConsensusDkgDealResponse:
//	//	return "MessageTypeConsensusDkgDealResponse"
//	//case MessageTypeConsensusDkgSigSets:
//	//	return "MessageTypeDkgSigSets"
//	//case MessageTypeConsensusDkgGenesisPublicKey:
//	//	return "MessageTypeConsensusDkgGenesisPublicKey"
//	case MessageTypeTermChangeRequest:
//		return "MessageTypeTermChangeRequest"
//	case MessageTypeTermChangeResponse:
//		return "MessageTypeTermChangeResponse"
//	case MessageTypeSecret:
//		return "MessageTypeSecret"
//
//	//case MessageTypeProposal:
//	//	return "MessageTypeProposal"
//	//case MessageTypePreVote:
//	//	return "MessageTypePreVote"
//	//case MessageTypePreCommit:
//	//	return "MessageTypePreCommit"
//
//	case MessageTypeOg01Length: //og01 length
//		return "MessageTypeOg01Length"
//
//		// Protocol messages belonging to og/02
//
//	case GetNodeDataMsg:
//		return "GetNodeDataMsg"
//	case NodeDataMsg:
//		return "NodeDataMsg"
//	case GetReceiptsMsg:
//		return "GetReceiptsMsg"
//	case MessageTypeOg02Length:
//		return "MessageTypeOg02Length"
//	case MessageTypeEncrypted:
//		return "MessageTypeEncrypted"
//	case MessageTypeSigned:
//		return "MessageTypeSigned"
//	default:
//		return fmt.Sprintf("unkown message type %d", mt)
//	}
//}

//
//func (mt BinaryMessageType) Code() p2p.MsgCodeType {
//	return p2p.MsgCodeType(mt)
//}
