package account

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"sync"
)

type SampleAccount struct {
	Id          int
	PrivateKey  crypto.PrivateKey
	PublicKey   crypto.PublicKey
	Address     types.Address
	nonce       uint64
	nonceInited bool
	mu          sync.RWMutex
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

func (s *SampleAccount) ConsumeNonce() (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.nonceInited {
		return 0, fmt.Errorf("nonce is not initialized. Query first")
	}
	s.nonce++
	return s.nonce, nil
}

func (s *SampleAccount) GetNonce() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.nonceInited {
		return 0, fmt.Errorf("nonce is not initialized. Query first")
	}
	return s.nonce, nil
}

func (s *SampleAccount) SetNonce(lastUsedNonce uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonce = lastUsedNonce
	s.nonceInited = true
}
