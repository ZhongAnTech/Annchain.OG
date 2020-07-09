package state

import (
	"fmt"
	ogtypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/utils/marshaller"
	"github.com/annchain/OG/common/math"
	"math/big"
)

const (
	MaxTokenName   = 20
	MaxTokenSymbol = 5
)

//go:generate msgp

//msgp:tuple TokenObject
type TokenObject struct {
	TokenID    int32
	Name       string
	Symbol     string
	Issuer     ogtypes.Address
	ReIssuable bool

	Issues    []*math.BigInt
	Destroyed bool

	db StateDBInterface
}

func NewTokenObject(tokenID int32, issuer ogtypes.Address, name, symbol string, reIssuable bool, fstIssue *math.BigInt, db StateDBInterface) *TokenObject {

	if len(name) > MaxTokenName {
		name = name[:MaxTokenName]
	}
	if len(symbol) > MaxTokenSymbol {
		symbol = symbol[:MaxTokenSymbol]
	}

	t := &TokenObject{}

	t.TokenID = tokenID
	t.Issuer = issuer
	t.Name = name
	t.Symbol = symbol
	t.ReIssuable = reIssuable
	t.Issues = []*math.BigInt{math.NewBigIntFromBigInt(fstIssue.Value)}
	t.Destroyed = false

	t.db = db

	return t
}

func (t *TokenObject) GetID() int32 {
	return t.TokenID
}

func (t *TokenObject) GetName() string {
	return t.Name
}

func (t *TokenObject) GetSymbol() string {
	return t.Symbol
}

func (t *TokenObject) CanReIssue() bool {
	return t.ReIssuable
}

func (t *TokenObject) AllIssues() []*math.BigInt {
	return t.Issues
}

func (t *TokenObject) OneIssue(term int) *math.BigInt {
	if len(t.Issues) <= term {
		return math.NewBigInt(0)
	}
	return t.Issues[term]
}

/**
Setters
*/

func (t *TokenObject) ReIssue(amount *math.BigInt) error {
	if t.Destroyed {
		return fmt.Errorf("token has been destroyed")
	}
	if !t.ReIssuable {
		return fmt.Errorf("token is not able to reissue")
	}
	t.db.AppendJournal(&reIssueChange{
		tokenID: t.TokenID,
	})
	t.Issues = append(t.Issues, amount)
	return nil
}

func (t *TokenObject) Destroy() {
	t.db.AppendJournal(&destroyChange{
		tokenID:       t.TokenID,
		prevDestroyed: t.Destroyed,
	})
	t.Destroyed = true
}

func (t *TokenObject) CopyRaw(tObj *TokenObject) {
	t.TokenID = tObj.TokenID
	t.Name = tObj.Name
	t.Symbol = tObj.Symbol
	t.Issuer = tObj.Issuer
	t.ReIssuable = tObj.ReIssuable
	t.Issues = tObj.Issues
	t.Destroyed = tObj.Destroyed
}

func (t *TokenObject) Encode() ([]byte, error) {
	return t.MarshalMsg()
}

func (t *TokenObject) Decode(b []byte) error {
	_, err := t.UnmarshalMsg(b)
	return err
}

/**
Marshaller part
 */

func (t *TokenObject) MarshalMsg() ([]byte, error) {
	var err error
	b := make([]byte, marshaller.HeaderSize)

	// int32 TokenID
	b = marshaller.AppendInt32(b, t.TokenID)
	// string Name
	b = marshaller.AppendString(b, t.Name)
	// string Symbol
	b = marshaller.AppendString(b, t.Symbol)
	// Address
	b, err = marshaller.AppendIMarshaller(b, t.Issuer)
	if err != nil {
		return b, err
	}
	// bool ReIssuable
	b = marshaller.AppendBool(b, t.ReIssuable)
	// []math.BigInt issues
	var btsArr [][]byte
	for _, bi := range t.Issues {
		btsArr = append(btsArr, bi.GetBytes())
	}
	b = marshaller.AppendBytesArray(b, btsArr)
	// bool Destroyed
	b = marshaller.AppendBool(b, t.Destroyed)

	return b, nil
}

func (t *TokenObject) UnmarshalMsg(b []byte) ([]byte, error) {
	b, _, err := marshaller.DecodeHeader(b)
	if err != nil {
		return nil, err
	}

	// TokenID
	t.TokenID, b, err = marshaller.ReadInt32(b)
	if err != nil {
		return nil, err
	}
	// Name
	t.Name, b, err = marshaller.ReadString(b)
	if err != nil {
		return nil, err
	}
	// Symbol
	t.Symbol, b, err = marshaller.ReadString(b)
	if err != nil {
		return nil, err
	}
	// Issuer
	t.Issuer, b, err = ogtypes.UnmarshalAddress(b)
	if err != nil {
		return nil, err
	}
	// ReIssuable
	t.ReIssuable, b, err = marshaller.ReadBool(b)
	if err != nil {
		return nil, err
	}
	// issues
	btss, b, err := marshaller.ReadBytesArray(b)
	if err != nil {
		return nil, err
	}
	t.Issues = make([]*math.BigInt, len(btss))
	for i, bts := range btss {
		t.Issues[i] = math.NewBigIntFromBigInt(big.NewInt(0).SetBytes(bts))
	}
	// Destroyed
	t.Destroyed, b, err = marshaller.ReadBool(b)
	if err != nil {
		return nil, err
	}

	return b, err
}
