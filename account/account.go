package account

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"go.uber.org/atomic"
	"sync"
)

type Account struct {
	Id          int
	PrivateKey  crypto.PrivateKey
	PublicKey   crypto.PublicKey
	Address     common.Address
	InitBalance uint64

	nonce       atomic.Uint64
	nonceInited bool
	mu          sync.RWMutex
}

func NewAccount(privateKeyHex string) *Account {
	s := &Account{}
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

func (s *Account) ConsumeNonce() (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.nonceInited {
		return 0, fmt.Errorf("nonce is not initialized. Query first")
	}
	s.nonce.Inc()
	return s.nonce.Load(), nil
}

func (s *Account) GetNonce() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.nonceInited {
		return 0, fmt.Errorf("nonce is not initialized. Query first")
	}
	return s.nonce.Load(), nil
}

func (s *Account) SetNonce(lastUsedNonce uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonce.Store(lastUsedNonce)
	s.nonceInited = true
}
