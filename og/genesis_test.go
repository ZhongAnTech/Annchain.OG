package og

import (
	"testing"
	
	"fmt"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/common/crypto"
)

func TestDefaultGenesis(t *testing.T) {
	t.Parallel()

	txCreator := TxCreator{
		Signer: crypto.CryptoTypeSecp256k1,
	}
	// TODO delete !
	pk, _ := crypto.PrivateKeyFromString("6f6720697320746865206265737420636861696e000000000000000000000000")
	seq := txCreator.NewUnsignedSequencer(0, []types.Hash{}, 0)
	sig := txCreator.Signer.Sign(pk, seq.SignatureTargets())
	seq.GetBase().Signature = sig.Bytes
	seq.GetBase().PublicKey = txCreator.Signer.PubKey(pk).Bytes

	addr := txCreator.Signer.Address(txCreator.Signer.PubKey(pk))
	balance := map[types.Address]*math.BigInt{}
	balance[addr] = math.NewBigInt(999999999999999999999)

	fmt.Printf("sig: %x", seq.GetBase().Signature)
	fmt.Printf("pub: %x", seq.GetBase().PublicKey)
	fmt.Printf("addr: %x", addr.ToBytes())

}
