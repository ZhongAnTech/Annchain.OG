package crypto

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
	"encoding/hex"
	"bytes"
)

func TestSigner(t *testing.T) {
	signer := SignerEd25519{}

	pub, priv, err := signer.RandomKeyPair()
	assert.NoError(t, err)

	fmt.Println(hex.Dump(pub.Bytes))
	fmt.Println(hex.Dump(priv.Bytes))
	address := signer.Address(pub)
	fmt.Println(hex.Dump(address.Bytes[:]))
	fmt.Println(signer.Address(pub).Hex())

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