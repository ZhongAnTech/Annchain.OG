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
	pk, priv, _ := signer.RandomKeyPair()

	b := []byte("foo")
	sig := signer.Sign(priv, b)
	if !signer.Verify(pk, sig, b) {
		t.Fatalf("vertfy failed")
	}

}
