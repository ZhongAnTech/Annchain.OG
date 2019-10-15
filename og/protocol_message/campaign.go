// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package protocol_message

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/consensus/vrf"
	"github.com/annchain/OG/types"
	"github.com/annchain/kyber/v3"
	"strings"

	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp

//msgp:tuple Campaign
type Campaign struct {
	TxBase
	DkgPublicKey []byte
	Vrf          vrf.VrfInfo
	Issuer       *common.Address
	dkgPublicKey kyber.Point
}

//msgp:tuple Campaigns
type Campaigns []*Campaign

func (c *Campaign) GetBase() *TxBase {
	return &c.TxBase
}

func (c *Campaign) Sender() common.Address {
	return *c.Issuer
}

func (tc *Campaign) GetSender() *common.Address {
	return tc.Issuer
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
		strings.Join(phashes, " ,"), c.Issuer, c.AccountNonce, hexutil.Encode(c.Signature), hexutil.Encode(c.PublicKey),
		c.DkgPublicKey, c.Vrf.PublicKey, c.Vrf.Message, c.Vrf.Vrf, c.Vrf.Proof)
}

func (c *Campaign) SignatureTargets() []byte {
	// add parents infornmation.
	w := types.NewBinaryWriter()

	w.Write(c.DkgPublicKey, c.Vrf.Vrf, c.Vrf.PublicKey, c.Vrf.Proof, c.Vrf.Message, c.AccountNonce)

	if !CanRecoverPubFromSig {
		w.Write(c.Issuer.Bytes)
	}

	return w.Bytes()
}

func (c *Campaign) String() string {
	if c.GetSender() == nil {
		return fmt.Sprintf("%s-[nil]-%d-%s-Cp", c.TxBase.String(), c.AccountNonce, hexutil.Encode(c.DkgPublicKey[:5]))
	}
	return fmt.Sprintf("%s-[%.10s]-%d-%s-Cp", c.TxBase.String(), c.Sender().String(), c.AccountNonce, hexutil.Encode(c.DkgPublicKey[:5]))
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

func (c *Campaign) RawTxi() RawTxi {
	return c.RawCampaign()
}

func (t *Campaign) SetSender(addr common.Address) {
	t.Issuer = &addr
}

func (r *Campaigns) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}