package state_test

import (
	"testing"

	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/types"
)

var (
	testAddress = "0x0b5d53f433b7e4a4f853a01e987f977497dda262"
)

func TestSerialization(t *testing.T) {
	t.Parallel()

	testnonce := uint64(123456)
	testblc := int64(666)

	stdb := newTestStateDB(t)
	stdb.CreateAccount(types.HexToAddress(testAddress))

	addr := types.HexToAddress(testAddress)
	stobj := stdb.GetStateObject(addr)
	stobj.SetNonce(testnonce)
	stobj.SetBalance(math.NewBigInt(testblc))

	b, err := stobj.Encode()
	if err != nil {
		t.Errorf("encode state object meet error: %v", err)
	}
	var newstobj state.StateObject
	err = newstobj.Decode(b, stdb)
	if err != nil {
		t.Errorf("decode state object error: %v", err)
	}
	newnonce := newstobj.GetNonce()
	if newnonce != testnonce {
		t.Errorf("nonce error, should be %d, but get %d", testnonce, newnonce)
	}
	newblc := newstobj.GetBalance()
	if newblc.GetInt64() != testblc {
		t.Errorf("balance error, should be %d, but get %d", testblc, newblc.GetInt64())
	}

}
