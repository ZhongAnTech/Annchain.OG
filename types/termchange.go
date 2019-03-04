package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp

//msgp:tuple TermChange
type TermChange struct {
	TxBase
	PkBls  []byte
	SigSet []*SigSet
	Issuer Address
}

//msgp:tuple SigSet
type SigSet struct {
	PublicKey []byte
	Signature []byte
}

//msgp:tuple TermChanges
type TermChanges []*TermChange

func (tc *TermChange) GetBase() *TxBase {
	return &tc.TxBase
}

func (tc *TermChange) Sender() Address {
	return tc.Issuer
}

func (tc *TermChange) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *TermChange:
		if tc.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (tc *TermChange) Dump() string {
	var phashes []string
	for _, p := range tc.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	var sigs []string
	for _, v := range tc.SigSet {
		sigs = append(sigs, fmt.Sprintf("[pubkey: %x, sig: %x]", v.PublicKey, v.Signature))
	}
	return fmt.Sprintf("hash: %s, pHash: [%s], issuer: %s, nonce: %d , signatute: %s, pubkey %s ,"+
		"PkBls: %x, sigs: [%s]", tc.Hash.Hex(),
		strings.Join(phashes, ", "), tc.Issuer.Hex(), tc.AccountNonce, hexutil.Encode(tc.Signature), hexutil.Encode(tc.PublicKey),
		tc.PkBls, strings.Join(sigs, ", "))
}

func (tc *TermChange) SignatureTargets() []byte {
	// TODO
	// add parents infornmation.
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, tc.AccountNonce))
	panicIfError(binary.Write(&buf, binary.BigEndian, tc.Issuer.Bytes))
	panicIfError(binary.Write(&buf, binary.BigEndian, tc.PkBls))
	for _, sig := range tc.SigSet {
		panicIfError(binary.Write(&buf, binary.BigEndian, sig.PublicKey))
		panicIfError(binary.Write(&buf, binary.BigEndian, sig.Signature))
	}
	return buf.Bytes()
}

func (tc *TermChange) String() string {
	return fmt.Sprintf("%s-[%.10s]-termChange", tc.TxBase.String(), tc.Issuer.String())
}

func (c TermChanges) String() string {
	var strs []string
	for _, v := range c {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (c *TermChange) RawTermChange() *RawTermChange {
	if c == nil {
		return nil
	}
	rc := &RawTermChange{
		TxBase: c.TxBase,
		PkBls:  c.PkBls,
		SigSet: c.SigSet,
	}
	return rc
}

func (cs TermChanges) RawTermChanges() RawTermChanges {
	if len(cs) == 0 {
		return nil
	}
	var rawTs RawTermChanges
	for _, v := range cs {
		rasSeq := v.RawTermChange()
		rawTs = append(rawTs, rasSeq)
	}
	return rawTs
}

func (c *TermChange) RawTxi() RawTxi {
	return c.RawTermChange()
}
