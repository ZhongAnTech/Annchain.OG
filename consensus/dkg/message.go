package dkg

import (
	"fmt"
	"github.com/annchain/OG/types"
	dkg "github.com/annchain/kyber/v3/share/dkg/pedersen"
)

//go:generate msgp

//msgp:tuple DkgMessageType
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
		return "DkgUnknown"
	}
}

type Signable interface {
	SignatureTargets() []byte
}

type DkgMessage interface {
	Signable
	GetType() DkgMessageType
	ProvideHeight() uint64
	String() string
}

//msgp:tuple DkgBasicInfo
type DkgBasicInfo struct {
	//TermId uint32
	Height uint64
	//PublicKey PublicKeyMarshallable
}

func (d *DkgBasicInfo) ProvideHeight() uint64 {
	return d.Height
}

//msgp:tuple MessageDkgGenesisPublicKey
type MessageDkgGenesisPublicKey struct {
	DkgBasicInfo
	PublicKeyBytes []byte
	//PublicKey    []byte
	//Signature    []byte
}

func (z *MessageDkgGenesisPublicKey) GetType() DkgMessageType {
	return DkgMessageTypeGenesisPublicKey
}

func (z *MessageDkgGenesisPublicKey) String() string {
	return "DkgMessageTypeGenesisPublicKey"
}

func (z *MessageDkgGenesisPublicKey) SignatureTargets() []byte {
	return z.PublicKeyBytes
}

//msgp:tuple MessageDkgSigSets
type MessageDkgSigSets struct {
	DkgBasicInfo
	PkBls []byte
}

func (z *MessageDkgSigSets) GetType() DkgMessageType {
	return DkgMessageTypeSigSets
}

func (z *MessageDkgSigSets) String() string {
	return "DkgMessageTypeSigSets"
}

func (m *MessageDkgSigSets) SignatureTargets() []byte {
	w := types.NewBinaryWriter()
	w.Write(m.PkBls)
	return w.Bytes()
}

//msgp:tuple MessageDkgDeal
type MessageDkgDeal struct {
	DkgBasicInfo
	//Id   uint32
	Data []byte
}

func (z *MessageDkgDeal) GetType() DkgMessageType {
	return DkgMessageTypeDeal
}

func (m *MessageDkgDeal) GetDeal() (*dkg.Deal, error) {
	var d dkg.Deal
	_, err := d.UnmarshalMsg(m.Data)
	return &d, err
}

func (m *MessageDkgDeal) SignatureTargets() []byte {
	w := types.NewBinaryWriter()
	d := m.Data
	//w.Write(d, m.Id)
	w.Write(d)
	return w.Bytes()
}

func (m MessageDkgDeal) String() string {
	//var pkstr string
	//if len(m.PublicKey) > 10 {
	//	pkstr = hexutil.Encode(m.PublicKey[:5])
	//}
	return "dkg " + fmt.Sprintf("len %d  height %d", len(m.Data), m.Height) //  + " pk-" + pkstr
}

//msgp:tuple MessageDkgDealResponse
type MessageDkgDealResponse struct {
	DkgBasicInfo
	//MyIndex   uint32
	Data []byte
	//PublicKey []byte
	//Signature []byte
	//SessionId uint64
}

func (m MessageDkgDealResponse) GetType() DkgMessageType {
	return DkgMessageTypeDealResponse
}

func (m MessageDkgDealResponse) String() string {
	//var pkstr string
	//if len(m.PublicKey) > 10 {
	//	pkstr = hexutil.Encode(m.PublicKey[:5])
	//}
	return "dkgresponse " //+ fmt.Sprintf(" pk %s , len %d  tid %d", hexutil.Encode(m.PublicKey), len(m.Data), m.TermId) // + " pk-" + pkstr
}

func (m *MessageDkgDealResponse) SignatureTargets() []byte {
	d := m.Data
	w := types.NewBinaryWriter()
	//w.Write(d, m.Id)
	w.Write(d)
	return w.Bytes()
}

func (m *MessageDkgDealResponse) GetResponse() (response *dkg.Response, err error) {
	var d dkg.Response
	_, err = d.UnmarshalMsg(m.Data)
	return &d, err
}
