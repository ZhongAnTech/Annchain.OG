package account

import (
	"fmt"
	"testing"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
)

func TestSignature(t *testing.T) {
	acc := NewAccount(fmt.Sprintf("0x0170E6B713CD32904D07A55B3AF5784E0B23EB38589EBF975F0AB89E6F8D786F%02d", 00))

	tx := &types.Tx{
		From:  acc.Address,
		To:    types.HexToAddress("0x1234567812345678"),
		Value: math.NewBigInt(0xAABBCCDDEEFF),
		TxBase: types.TxBase{
			PublicKey: acc.PublicKey.Bytes,
			Height:    10,
		},
	}
	signer := &crypto.SignerSecp256k1{}

	beforeSign := tx.SignatureTargets()
	fmt.Println(hexutil.Encode(beforeSign))
	sig := signer.Sign(acc.PrivateKey, tx.SignatureTargets())
	fmt.Println(hexutil.Encode(sig.Bytes))
}

func TestSampleAccounts(t *testing.T) {
	MaxAccountCount := 255
	for i := 0; i < MaxAccountCount; i++ {
		acc := NewAccount(fmt.Sprintf("0x0170E6B713CD32904D07A55B3AF5784E0B23EB38589EBF975F0AB89E6F8D786F%02X", i))
		fmt.Println(fmt.Sprintf("account address: %s, pubkey: %s, privkey: %s", acc.Address.String(), acc.PublicKey.String(), acc.PrivateKey.String()))
	}

}

// time="2018-11-07T15:26:07+08:00" level=info msg="Sample Account" add=0x0b5d53f433b7e4a4f853a01e987f977497dda262 priv=0x0170E6B713CD32904D07A55B3AF5784E0B23EB38589EBF975F0AB89E6F8D786F00

// addr=0x0b5d53f433b7e4a4f853a01e987f977497dda262
// priv=0x70E6B713CD32904D07A55B3AF5784E0B23EB38589EBF975F0AB89E6F8D786F00
// sig content=0x00000000000000000b5d53f433b7e4a4f853a01e987f977497dda2621234567812345678000000000000000000000000aabbccddeeff
// sig=0x3045022100c3bddcbb80245eaac2ad79c6aaff46a44c9824fd3925ff779689c8e7d1c541f80220594dfa2bf76c143b6fb7c7be46cfd59e49d959d84fc0a22bd48faf63f87137db

//0x00000000000000000b5d53f433b7e4a4f853a01e987f977497dda2621234567812345678000000000000000000000000aabbccddeeff
//0x3045022100c3bddcbb80245eaac2ad79c6aaff46a44c9824fd3925ff779689c8e7d1c541f80220594dfa2bf76c143b6fb7c7be46cfd59e49d959d84fc0a22bd48faf63f87137db
