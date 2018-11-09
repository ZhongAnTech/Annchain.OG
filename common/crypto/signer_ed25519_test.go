package crypto

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/stretchr/testify/assert"
	"testing"
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

func TestSignerEd25519_Sign(t *testing.T) {
	data := common.FromHex("0x000000000000000099fa25cbb3d8fdd1ec29aa499cf651972f0ee67b0000000000000001")
	signer := &SignerEd25519{}
	//privKey,_:= PrivateKeyFromString("0x009d9d0fe5e9ef0bb3bb4934db878688500fd0fd8e026c1ff1249b7e268c8a363aa7d45d13a5accb299dc7fe0f3b5fb0e9526b67008f7ead02c51c7b1f5a1d7b00")
	_, privKey, _ := signer.RandomKeyPair()
	sig := signer.Sign(privKey, data)
	fmt.Printf("%x\n", sig.Bytes[:])
	ok := signer.Verify(signer.PubKey(privKey), sig, data)
	if !ok {
		t.Fatal(ok)
	}
}
