package account

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"go.uber.org/atomic"
	"sync"
)

type SampleAccount struct {
	Id          int
	PrivateKey  crypto.PrivateKey
	PublicKey   crypto.PublicKey
	Address     types.Address
	nonce       atomic.Uint64
	nonceInited bool
	mu          sync.RWMutex
}

func NewAccount(privateKeyHex string) *SampleAccount {
	s := &SampleAccount{}
	pv, err := crypto.PrivateKeyFromString(privateKeyHex)
	if err != nil {
		panic(err)
	}
	signer := crypto.NewSigner(pv.Type)
	s.PrivateKey = pv
	s.PublicKey = signer.PubKey(pv)
	s.Address = signer.Address(s.PublicKey)
	//logrus.WithField("add", s.Address.String()).WithField("priv", privateKeyHex).Trace("Sample Account")

	return s
}

func (s *SampleAccount) ConsumeNonce() (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.nonceInited {
		return 0, fmt.Errorf("nonce is not initialized. Query first")
	}
	s.nonce.Inc()
	return s.nonce.Load(), nil
}

func (s *SampleAccount) GetNonce() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.nonceInited {
		return 0, fmt.Errorf("nonce is not initialized. Query first")
	}
	return s.nonce.Load(), nil
}

func (s *SampleAccount) SetNonce(lastUsedNonce uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonce.Store(lastUsedNonce)
	s.nonceInited = true
}
