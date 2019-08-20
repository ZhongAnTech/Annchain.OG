package dkg

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/msg"
	dkg "github.com/annchain/kyber/v3/share/dkg/pedersen"
)

//go:generate msgp

//msgp:tuple BftMessageType
type DkgMessageType uint16

const (
	DkgMessageTypeDeal DkgMessageType = iota + 200
	DkgMessageTypeDealResponse
	DkgMessageTypeSigSets
	DkgMessageTypeGenesisPublicKey
)

func (m DkgMessageType) String() string {
	switch m {
	case DkgMessageTypeDeal:
		return "DkgMessageTypeDeal"
	case DkgMessageTypeDealResponse:
		return "DkgMessageTypeDealResponse"
	case DkgMessageTypeSigSets:
		return "DkgMessageTypeSigSets"
	case DkgMessageTypeGenesisPublicKey:
		return "DkgMessageTypeGenesisPublicKey"
	default:
		return "BFTUnknown"
	}
}

type Signable interface {
	msg.MsgpMember
	SignatureTargets() []byte
}

//msgp:tuple DkgMessage
type DkgMessage struct {
	Type    DkgMessageType
	Payload Signable
}

func (m *DkgMessage) String() string {
	return fmt.Sprintf("%s %+v", m.Type.String(), m.Payload)
}

//msgp:tuple BftBasicInfo
type DkgBasicInfo struct {
	TermId    uint64
	PublicKey crypto.PublicKey
}

//msgp:tuple MessageDkgGenesisPublicKey
type MessageDkgGenesisPublicKey struct {
	DkgBasicInfo
	DkgPublicKey []byte
	PublicKey    []byte
	Signature    []byte
}

func (m *MessageDkgGenesisPublicKey) SignatureTargets() []byte {
	return m.DkgPublicKey
}

func (m *MessageDkgGenesisPublicKey) String() string {
	return fmt.Sprintf("DkgGenesisPublicKey  len %d ", len(m.DkgPublicKey))
}

//msgp:tuple MessageDkgSigSets
type MessageDkgSigSets struct {
	DkgBasicInfo
	PkBls []byte
}

func (m *MessageDkgSigSets) SignatureTargets() []byte {
	w := types.NewBinaryWriter()
	w.Write(m.PkBls)
	return w.Bytes()
}

func (m *MessageDkgSigSets) String() string {
	return "dkgSigsets" + fmt.Sprintf("len %d", len(m.PkBls))
}

//msgp:tuple MessageDkgDeal
type MessageDkgDeal struct {
	DkgBasicInfo
	Id   uint32
	Data []byte
}

func (m *MessageDkgDeal) GetDeal() (*dkg.Deal, error){
	var d dkg.Deal
	_, err := d.UnmarshalMsg(m.Data)
	return &d, err
}

func (m *MessageDkgDeal) SignatureTargets() []byte {
	w := types.NewBinaryWriter()
	d := m.Data
	w.Write(d, m.Id)
	return w.Bytes()
}

func (m MessageDkgDeal) String() string {
	//var pkstr string
	//if len(m.PublicKey) > 10 {
	//	pkstr = hexutil.Encode(m.PublicKey[:5])
	//}
	return "dkg " + fmt.Sprintf(" id %d , len %d  tid %d", m.Id, len(m.Data), m.TermId) //  + " pk-" + pkstr
}

//msgp:tuple MessageDkgDealResponse
type MessageDkgDealResponse struct {
	DkgBasicInfo
	//Id   uint32
	Data []byte
	//PublicKey []byte
	//Signature []byte
	//TermId uint64
}

func (m MessageDkgDealResponse) String() string {
	//var pkstr string
	//if len(m.PublicKey) > 10 {
	//	pkstr = hexutil.Encode(m.PublicKey[:5])
	//}
	return "dkgresponse " + fmt.Sprintf(" pk %s , len %d  tid %d", m.PublicKey.String(), len(m.Data), m.TermId) // + " pk-" + pkstr
}

func (m *MessageDkgDealResponse) SignatureTargets() []byte {
	d := m.Data
	w := types.NewBinaryWriter()
	w.Write(d, m.Id)
	w.Write(d)
	return w.Bytes()
}
