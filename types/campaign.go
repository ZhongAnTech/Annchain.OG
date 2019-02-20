package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp
//msgp:tuple Campaign

type Campaign struct {
	TxBase
	DkgPublicKey  []byte
	Vrf       VrfInfo
	Issuer Address
}

type VrfInfo struct  {
	Message []byte
	Proof   []byte
	PublicKey []byte
	Vrf        []byte
}

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
	// TODO
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


func (c*Campaign)String()string  {
	return fmt.Sprintf("%s-[%s]-campain",c.TxBase.String(),c.Issuer.String())
}
