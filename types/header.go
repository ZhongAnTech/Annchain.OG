package types

type SequencerHeader struct {
	hash Hash
	id   uint64
}

func (s *SequencerHeader) SequencerId() uint64 {
	return s.id
}

func (s *SequencerHeader) Hash() Hash {
	return s.hash
}

func (s *SequencerHeader) Id() uint64 {
	return s.id
}

func NewSequencerHead(hash Hash, id uint64) *SequencerHeader {
	return &SequencerHeader{
		hash: hash,
		id:   id,
	}
}

func SeqsToHeaders(seqs []*Sequencer) []*SequencerHeader {
	if len(seqs) == 0 {
		return nil
	}
	var headers []*SequencerHeader
	for _, v := range seqs {
		head := NewSequencerHead(v.Hash, v.Id)
		headers = append(headers, head)
	}
	return headers
}
