package state

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
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
	Issuer     common.Address
	ReIssuable bool

	Issues    []*math.BigInt
	Destroyed bool

	db StateDBInterface
}

func NewTokenObject(tokenID int32, issuer common.Address, name, symbol string, reIssuable bool, fstIssue *math.BigInt, db StateDBInterface) *TokenObject {

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
	return t.MarshalMsg(nil)
}

func (t *TokenObject) Decode(b []byte) error {
	_, err := t.UnmarshalMsg(b)
	return err
}
