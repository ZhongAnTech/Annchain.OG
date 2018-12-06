package crypto

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignerSecp(t *testing.T) {
	signer := SignerSecp256k1{}

	pub, priv, err := signer.RandomKeyPair()
	assert.NoError(t, err)

	fmt.Println(hex.Dump(pub.Bytes))
	fmt.Println(hex.Dump(priv.Bytes))
	address := signer.Address(pub)
	fmt.Println(hex.Dump(address.Bytes[:]))
	fmt.Println(signer.Address(pub).Hex())

	fmt.Printf("%x\n", priv.Bytes[:])
	fmt.Printf("%x\n", pub.Bytes[:])
	fmt.Printf("%x\n", address.Bytes[:])

	pub2 := signer.PubKey(priv)
	fmt.Println(hex.Dump(pub2.Bytes))
	assert.True(t, bytes.Equal(pub.Bytes, pub2.Bytes))

	content := []byte("This is a test")
	sig := signer.Sign(priv, content)
	fmt.Println(hex.Dump(sig.Bytes))

	assert.True(t, signer.Verify(pub2, sig, content))

	content[0] = 0x88
	assert.False(t, signer.Verify(pub2, sig, content))

}

func TestSignerNewPrivKey(t *testing.T) {
	t.Parallel()

	signer := SignerSecp256k1{}
	// priv, _ := PrivateKeyFromString("0x0170e6b713cd32904d07a55b3af5784e0b23eb38589ebf975f0ab89e6f8d786f00")
	pk, priv, _ := signer.RandomKeyPair()

	b := []byte("foo")
	sig := signer.Sign(priv, b)

	// _, ecdsapub := ecdsabtcec.PrivKeyFromBytes(ecdsabtcec.S256(), priv.Bytes)
	// pubKeybyte := ethcrypto.FromECDSAPub((*ecdsa.PublicKey)(ecdsapub))
	// pk := PublicKey{
	// 	Type:  CryptoTypeSecp256k1,
	// 	Bytes: pubKeybyte,
	// }

	// pk := signer.PubKey(priv)

	sigstr := fmt.Sprintf("%x", sig.Bytes)
	if sigstr != "c3bddcbb80245eaac2ad79c6aaff46a44c9824fd3925ff779689c8e7d1c541f8594dfa2bf76c143b6fb7c7be46cfd59e49d959d84fc0a22bd48faf63f87137db" {
		t.Logf("sig is: %x", sig.Bytes)
		t.Logf("self verify: %v", signer.Verify(pk, sig, b))
		t.Fatalf("not same signature, %s", sigstr)
	}
}
