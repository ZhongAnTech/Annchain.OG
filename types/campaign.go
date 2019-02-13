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
	PkDkg  []byte
	PkVrf  []byte
	Data   []byte
	Vrf    []byte
	Proof  []byte
	Issuer Address
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
		c.PkDkg, c.PkVrf, c.Data, c.Vrf, c.Proof)
}

func (c *Campaign) SignatureTargets() []byte {
	// TODO
	// add parents infornmation.
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, c.AccountNonce))
	panicIfError(binary.Write(&buf, binary.BigEndian, c.Issuer.Bytes))

	return buf.Bytes()
}
