package syncer

import (
	"fmt"
	"github.com/annchain/OG/common"
	"strings"
	"sync"
)

type SequencerCache struct {
	mu sync.RWMutex
	proposal map[common.Hash] []string
	hashes common.Hashes
	size int
}

func NewSequencerCache(size int) *SequencerCache {
	return &SequencerCache{
		proposal:make(map[common.Hash][]string),
		size:size,
	}
}

func (p *SequencerCache)Add( seqHash  common.Hash,peerId string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var peers []string
	var ok bool
	peers, ok = p.proposal[seqHash]
	if !ok {
		if len(p.proposal) > p.size {
			num := len(p.proposal) - p.size
			oldHash := p.hashes[:num]
			p.hashes = p.hashes[num:]
			for _, key :=  range oldHash {
				delete(p.proposal,key)
			}
		}
	}
	peers = append(peers,peerId)
	p.proposal[seqHash] = peers
	p.hashes = append(p.hashes,seqHash)
}

func (p*SequencerCache)GetPeer(seqHash common.Hash) string  {
	p.mu.RLock()
	defer p.mu.RUnlock()
	peers := p.proposal[seqHash]
	if len(peers) >0 {
		return peers[0]
	}
	return ""
}

func (s SequencerCache)String()string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var str string
	str += fmt.Sprintf("size %d",s.size)
	str +=" hashes:"
	for _, h :=range s.hashes {
		str+= " "+ h.String()
	}
	str +=" map:"
	for k, v :=range s.proposal {
		str+=  " key " + k.String()+" val " + strings.Join(v,",")
	}
	return str
}