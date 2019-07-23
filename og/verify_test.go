package og

import (
	"fmt"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"testing"
)

func TestVerifiedType_IsGraphVerified(t *testing.T) {
	tx := tx_types.RandomTx()
	fmt.Println(tx.IsVerified())
	tx.SetVerified(types.VerifiedFormat)
	fmt.Println(tx.IsVerified())
	fmt.Println(tx.IsVerified().IsGraphVerified(), tx.IsVerified().IsFormatVerified())
	tx.SetVerified(types.VerifiedGraph)
	fmt.Println(tx.IsVerified())
	fmt.Println(tx.IsVerified().IsGraphVerified(), tx.IsVerified().IsFormatVerified())
	tx.SetVerified(0)
	fmt.Println(tx.IsVerified())
	fmt.Println(tx.IsVerified().IsGraphVerified(), tx.IsVerified().IsFormatVerified())
	tx.SetVerified(types.VerifiedGraph)
	fmt.Println(tx.IsVerified())
	fmt.Println(tx.IsVerified().IsGraphVerified(), tx.IsVerified().IsFormatVerified())
	tx.SetVerified(types.VerifiedFormat)
	fmt.Println(tx.IsVerified())
	fmt.Println(tx.IsVerified().IsGraphVerified(), tx.IsVerified().IsFormatVerified())
}
