package consensus

import (
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/sirupsen/logrus"
	"sync"
)

type BlsSignatureCollector struct {
	CommitteeProvider consensus_interface.CommitteeProvider
	Round             int64
	signatures        map[int]consensus_interface.Signature
	mu                sync.RWMutex
}

func (s *BlsSignatureCollector) Collected() bool {
	// TODO: Bls verify
	return len(s.signatures) >= s.CommitteeProvider.GetThreshold()
}

func (s *BlsSignatureCollector) InitDefault() {
	s.signatures = make(map[int]consensus_interface.Signature)
}

func (s *BlsSignatureCollector) GetThreshold() int {
	return s.CommitteeProvider.GetThreshold()
}

func (s *BlsSignatureCollector) GetCurrentCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.signatures)
}

func (s *BlsSignatureCollector) GetSignature(index int) (v consensus_interface.Signature, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok = s.signatures[index]
	return
}

func (s *BlsSignatureCollector) GetJointSignature() consensus_interface.JointSignature {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// TODO: bls recovery
	return nil
}

func (s *BlsSignatureCollector) Collect(sig consensus_interface.Signature, index int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	logrus.WithField("sig", sig).WithField("index", index).WithField("round", s.Round).Trace("signature collected")
	s.signatures[index] = sig
	// TODO: bls enrich
}

func (s *BlsSignatureCollector) Has(key int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.signatures[key]
	return ok
}
