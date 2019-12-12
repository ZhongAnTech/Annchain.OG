package campaign

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/vrf"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/types/msg"
	"strings"
)

//go:generate msgp

const (
	MessageTypeCampaign msg.BinaryMessageType = iota + 300
	MessageTypeTermChange
	MessageTypeTermChangeRequest
	MessageTypeTermChangeResponse
)

//msgp:tuple RawCampaign
type RawCampaign struct {
	types.TxBase
	DkgPublicKey []byte
	Vrf          vrf.VrfInfo
}

//msgp:tuple RawTermChange
type RawTermChange struct {
	types.TxBase
	TermId uint64
	PkBls  []byte
	SigSet []*SigSet
}

func (t *RawTermChange) String() string {
	return fmt.Sprintf("%s-%d_%d-RawTC", t.TxBase.String(), t.AccountNonce, t.Height)
}

func (t *RawCampaign) String() string {
	return fmt.Sprintf("%s-%d_%d-RawCP", t.TxBase.String(), t.AccountNonce, t.Height)
}

func (rc *RawCampaign) Campaign() *Campaign {
	if rc == nil {
		return nil
	}
	cp := &Campaign{
		TxBase:       rc.TxBase,
		DkgPublicKey: rc.DkgPublicKey,
		Vrf:          rc.Vrf,
	}
	if !archive.CanRecoverPubFromSig {
		addr := crypto.Signer.AddressFromPubKeyBytes(rc.PublicKey)
		cp.Issuer = &addr
	}
	return cp
}

func (r *RawTermChange) TermChange() *TermChange {
	if r == nil {
		return nil
	}
	t := &TermChange{
		TxBase: r.TxBase,
		PkBls:  r.PkBls,
		SigSet: r.SigSet,
		TermID: r.TermId,
	}
	if !archive.CanRecoverPubFromSig {
		addr := crypto.Signer.AddressFromPubKeyBytes(r.PublicKey)
		t.Issuer = &addr
	}
	return t
}
func (t *RawTermChange) Txi() types.Txi {
	return t.TermChange()
}

func (t *RawCampaign) Txi() types.Txi {
	return t.Campaign()
}

//msgp:tuple RawCampaigns
type RawCampaigns []*RawCampaign

//msgp:tuple RawTermChanges
type RawTermChanges []*RawTermChange

func (r RawCampaigns) Campaigns() Campaigns {
	if len(r) == 0 {
		return nil
	}
	var cs Campaigns
	for _, v := range r {
		c := v.Campaign()
		cs = append(cs, c)
	}
	return cs
}

func (r RawTermChanges) TermChanges() TermChanges {
	if len(r) == 0 {
		return nil
	}
	var cs TermChanges
	for _, v := range r {
		c := v.TermChange()
		cs = append(cs, c)
	}
	return cs
}

func (r RawTermChanges) String() string {
	var strs []string
	for _, v := range r {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (r RawCampaigns) String() string {
	var strs []string
	for _, v := range r {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (r RawTermChanges) Txis() types.Txis {
	if len(r) == 0 {
		return nil
	}
	var cs types.Txis
	for _, v := range r {
		c := v.TermChange()
		cs = append(cs, c)
	}
	return cs
}

func (r RawCampaigns) Txis() types.Txis {
	if len(r) == 0 {
		return nil
	}
	var cs types.Txis
	for _, v := range r {
		c := v.Campaign()
		cs = append(cs, c)
	}
	return cs
}

func (r *RawCampaigns) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}

func (r *RawTermChanges) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}

//msgp:tuple MessageCampaign
type MessageCampaign struct {
	RawCampaign *RawCampaign
}

func (m *MessageCampaign) GetType() msg.BinaryMessageType {
	return MessageTypeCampaign
}

func (m *MessageCampaign) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageCampaign) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageCampaign) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageCampaign) String() string {
	return m.RawCampaign.String()
}

//msgp:tuple MessageTermChange
type MessageTermChange struct {
	RawTermChange *RawTermChange
}

func (m *MessageTermChange) GetType() msg.BinaryMessageType {
	return MessageTypeTermChange
}

func (m *MessageTermChange) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageTermChange) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageTermChange) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageTermChange) String() string {
	return m.RawTermChange.String()
}

//msgp:tuple MessageTermChangeRequest
type MessageTermChangeRequest struct {
	Id uint32
}

func (m *MessageTermChangeRequest) GetType() msg.BinaryMessageType {
	return MessageTypeTermChangeRequest
}

func (m *MessageTermChangeRequest) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageTermChangeRequest) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageTermChangeRequest) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageTermChangeRequest) String() string {
	return fmt.Sprintf("requst id %d ", m.Id)
}

//msgp:tuple MessageTermChangeResponse
type MessageTermChangeResponse struct {
	TermChange *TermChange
	Id         uint32
}

func (m *MessageTermChangeResponse) GetType() msg.BinaryMessageType {
	return MessageTypeTermChangeResponse
}

func (m *MessageTermChangeResponse) GetData() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *MessageTermChangeResponse) ToBinary() msg.BinaryMessage {
	return msg.BinaryMessage{
		Type: m.GetType(),
		Data: m.GetData(),
	}
}

func (m *MessageTermChangeResponse) FromBinary(bs []byte) error {
	_, err := m.UnmarshalMsg(bs)
	return err
}

func (m *MessageTermChangeResponse) String() string {
	return fmt.Sprintf("requst id %d , %v ", m.Id, m.TermChange)
}
