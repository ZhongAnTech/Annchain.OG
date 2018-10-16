package account

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"sync"
)

type SampleAccount struct {
	Id         int
	PrivateKey crypto.PrivateKey
	PublicKey  crypto.PublicKey
	Address    types.Address
	nonce      uint64
	mu         sync.RWMutex
}

func NewAccount(privateKeyHex string) SampleAccount {
	signer := &crypto.SignerSecp256k1{}

	s := SampleAccount{}
	pv, err := crypto.PrivateKeyFromString(privateKeyHex)
	if err != nil {
		panic(err)
	}
	s.PrivateKey = pv
	s.PublicKey = signer.PubKey(pv)
	s.Address = signer.Address(s.PublicKey)
	return s
}

func (s *SampleAccount) ConsumeNonce() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonce++
	return s.nonce
}

func (s *SampleAccount) GetNonce() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nonce
}

func (s *SampleAccount) SetNonce(value uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonce = value
}
