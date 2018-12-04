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
	priv, _ := PrivateKeyFromString("0170e6b713cd32904d07a55b3af5784e0b23eb38589ebf975f0ab89e6f8d786f00")

	b, _ := hex.DecodeString("0x00000000000000000b5d53f433b7e4a4f853a01e987f977497dda2621234567812345678000000000000000000000000aabbccddeeff")
	sig := signer.Sign(priv, b)

	sigstr := fmt.Sprintf("%x", sig.Bytes)
	if sigstr != "c3bddcbb80245eaac2ad79c6aaff46a44c9824fd3925ff779689c8e7d1c541f8bf76c143b6fb7c7be46cfd59e49d959d84fc0a22bd48faf63f87137db" {
		t.Fatalf("not same signature, %s", sigstr)
	}
}
