package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/sha3"
	"math/rand"
	"strings"
)

//go:generate msgp

//msgp:tupple Archive
type Archive struct {
	TxBase
	Data []byte `json:"data"`
}

//msgp:tuple Campaigns
type Archives []*Archive

func (a *Archive) GetBase() *TxBase {
	return &a.TxBase
}

func (a *Archive) Sender() Address {
	panic("not implemented")
	return Address{}
	//return  &Address{}
}

func (c *Archive) Compare(tx Txi) bool {
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

func (c *Archive) Dump() string {
	var phashes []string
	for _, p := range c.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	return fmt.Sprintf("hash: %s, pHash: [%s] , nonce: %d  ,Data: %x", c.Hash.Hex(),
		strings.Join(phashes, " ,"), c.AccountNonce, c.Data)
}

func (a *Archive) SignatureTargets() []byte {
	// add parents infornmation.
	panic("not inplemented")
}

func (a *Archive) String() string {
	return fmt.Sprintf("%s-%d-Ac", a.TxBase.String(), a.AccountNonce)
}

func (as Archives) String() string {
	var strs []string
	for _, v := range as {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (a *Archive) RawArchive() *RawArchive {
	if a == nil {
		return nil
	}
	ra := RawArchive{
		Archive: *a,
	}
	return &ra
}

func (cs Archives) RawArchives() RawArchives {
	if len(cs) == 0 {
		return nil
	}
	var rawCps RawArchives
	for _, v := range cs {
		rasSeq := v.RawArchive()
		rawCps = append(rawCps, rasSeq)
	}
	return rawCps
}

func (c *Archive) RawTxi() RawTxi {
	return c.RawArchive()
}

func RandomArchive() *Archive {
	return &Archive{TxBase: TxBase{
		Hash:        randomHash(),
		Height:      uint64(rand.Int63n(1000)),
		ParentsHash: Hashes{randomHash(), randomHash()},
		Type:        TxBaseTypeArchive,
		//AccountNonce: uint64(rand.Int63n(50000)),
		Weight: uint64(rand.Int31n(2000)),
	},
		Data: randomHash().ToBytes(),
	}
}

func (t *Archive) CalcTxHash() (hash Hash) {
	var buf bytes.Buffer

	for _, ancestor := range t.ParentsHash {
		panicIfError(binary.Write(&buf, binary.BigEndian, ancestor.Bytes))
	}
	// do not use Height to calculate tx hash.
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Weight))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Data))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.CalcMinedHash().Bytes))

	result := sha3.Sum256(buf.Bytes())
	hash.MustSetBytes(result[0:], PaddingNone)
	return
}
