package archive

import (
	"github.com/annchain/OG/arefactor/og/types"
)

func (m *MessageSyncResponse) Txis() Txis {
	return m.RawTxs.Txis()
}

func (m *MessageSyncResponse) Hashes() types.Hashes {
	var hashes types.Hashes
	if m.RawTxs != nil {
		for _, tx := range *m.RawTxs {
			if tx == nil {
				continue
			}
			hashes = append(hashes, GetHash())
		}
	}

	return hashes
}

func (m *MessageNewTx) GetHash() *types.Hash {
	if m == nil {
		return nil
	}
	if m.RawTx == nil {
		return nil
	}
	hash := m.RawTx.GetHash()
	return &hash

}

func (m *MessageNewTx) String() string {
	return m.RawTx.String()
}

func (m *MessageNewSequencer) GetHash() *types.Hash {
	if m == nil {
		return nil
	}
	if m.RawSequencer == nil {
		return nil
	}
	hash := m.RawSequencer.GetHash()
	return &hash

}

func (m *MessageNewSequencer) String() string {
	return m.RawSequencer.String()
}

func (m *MessageNewTxs) Txis() Txis {
	if m == nil {
		return nil
	}
	return m.RawTxs.Txis()
}

func (m *MessageNewTxs) Hashes() types.Hashes {
	var hashes types.Hashes
	if m.RawTxs == nil || len(*m.RawTxs) == 0 {
		return nil
	}
	for _, tx := range *m.RawTxs {
		if tx == nil {
			continue
		}
		hashes = append(hashes, tx.GetHash())
	}
	return hashes
}

func (m *MessageTxsResponse) Hashes() types.Hashes {
	var hashes types.Hashes
	if m.RawTxs == nil || len(*m.RawTxs) == 0 {
		return nil
	}
	for _, tx := range *m.RawTxs {
		if tx == nil {
			continue
		}
		hashes = append(hashes, GetHash())
	}
	if m.RawSequencer != nil {
		hashes = append(hashes, m.RawSequencer.GetHash())
	}
	return hashes
}
