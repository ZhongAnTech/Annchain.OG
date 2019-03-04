package types

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"strings"

	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp

//msgp:tuple Campaign
type Campaign struct {
	TxBase
	DkgPublicKey []byte
	Vrf          VrfInfo
	Issuer       Address
	dkgPublicKey kyber.Point
}

//msgp:tuple VrfInfo
type VrfInfo struct {
	Message   []byte
	Proof     []byte
	PublicKey []byte
	Vrf       []byte
}

//msgp:tuple Campaigns
type Campaigns []*Campaign

func (c *Campaign) GetBase() *TxBase {
	return &c.TxBase
}

func (c *Campaign) Sender() Address {
	return c.Issuer
}

func (c *Campaign) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *Campaign:
		if c.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (c *Campaign) Dump() string {
	var phashes []string
	for _, p := range c.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	return fmt.Sprintf("hash: %s, pHash: [%s], issuer: %s, nonce: %d , signatute: %s, pubkey %s ,"+
		"PkDkg: %x, PkVrf: %x, Data: %x, Vrf: %x, Proof: %x", c.Hash.Hex(),
		strings.Join(phashes, " ,"), c.Issuer.Hex(), c.AccountNonce, hexutil.Encode(c.Signature), hexutil.Encode(c.PublicKey),
		c.DkgPublicKey, c.Vrf.PublicKey, c.Vrf.Message, c.Vrf.Vrf, c.Vrf.Proof)
}

func (c *Campaign) SignatureTargets() []byte {
	// add parents infornmation.
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, c.DkgPublicKey))
	panicIfError(binary.Write(&buf, binary.BigEndian, c.Vrf.Vrf))
	panicIfError(binary.Write(&buf, binary.BigEndian, c.Vrf.PublicKey))
	panicIfError(binary.Write(&buf, binary.BigEndian, c.Vrf.Proof))
	panicIfError(binary.Write(&buf, binary.BigEndian, c.Vrf.Message))
	panicIfError(binary.Write(&buf, binary.BigEndian, c.AccountNonce))
	panicIfError(binary.Write(&buf, binary.BigEndian, c.Issuer.Bytes))

	return buf.Bytes()
}

func (c *Campaign) String() string {
	return fmt.Sprintf("%s-[%.10s]-%d-Cp", c.TxBase.String(), c.Sender().String(), c.AccountNonce)
}

func (c Campaigns) String() string {
	var strs []string
	for _, v := range c {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (c *Campaign) RawCampaign() *RawCampaign {
	if c == nil {
		return nil
	}
	rc := &RawCampaign{
		TxBase:       c.TxBase,
		DkgPublicKey: c.DkgPublicKey,
		Vrf:          c.Vrf,
	}
	return rc
}

func (cs Campaigns) RawCampaigns() RawCampaigns {
	if len(cs) == 0 {
		return nil
	}
	var rawCps RawCampaigns
	for _, v := range cs {
		rasSeq := v.RawCampaign()
		rawCps = append(rawCps, rasSeq)
	}
	return rawCps
}

func (c *Campaign) GetDkgPublicKey() kyber.Point {
	return c.dkgPublicKey
}

func (c *Campaign) UnmarshalDkgKey(unmarshalFunc func(b []byte) (kyber.Point, error)) error {
	p, err := unmarshalFunc(c.DkgPublicKey)
	if err != nil {
		return err
	}
	c.dkgPublicKey = p
	return nil
}

func (c *Campaign) MarshalDkgKey() error {
	d, err := c.dkgPublicKey.MarshalBinary()
	if err != nil {
		return nil
	}
	c.DkgPublicKey = d
	return nil
}

func (v *VrfInfo) String() string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("Msg:%s, vrf :%s , proof :%s, pubKey :%s", hex.EncodeToString(v.Message), hex.EncodeToString(v.Vrf),
		hex.EncodeToString(v.Proof), hex.EncodeToString(v.PublicKey))
}

func (c *Campaign) RawTxi() RawTxi {
	return c.RawCampaign()
}
