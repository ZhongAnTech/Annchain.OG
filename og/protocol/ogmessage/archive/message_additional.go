package archive

import (
	"github.com/annchain/OG/common"
)

func (m *MessageSyncResponse) Txis() Txis {
	return m.RawTxs.Txis()
}

func (m *MessageSyncResponse) Hashes() common.Hashes {
	var hashes common.Hashes
	if m.RawTxs != nil {
		for _, tx := range *m.RawTxs {
			if tx == nil {
				continue
			}
			hashes = append(hashes, GetTxHash())
		}
	}

	return hashes
}

func (m *MessageNewTx) GetHash() *common.Hash {
	if m == nil {
		return nil
	}
	if m.RawTx == nil {
		return nil
	}
	hash := m.RawTx.GetTxHash()
	return &hash

}

func (m *MessageNewTx) String() string {
	return m.RawTx.String()
}

func (m *MessageNewSequencer) GetHash() *common.Hash {
	if m == nil {
		return nil
	}
	if m.RawSequencer == nil {
		return nil
	}
	hash := m.RawSequencer.GetTxHash()
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

func (m *MessageNewTxs) Hashes() common.Hashes {
	var hashes common.Hashes
	if m.RawTxs == nil || len(*m.RawTxs) == 0 {
		return nil
	}
	for _, tx := range *m.RawTxs {
		if tx == nil {
			continue
		}
		hashes = append(hashes, tx.GetTxHash())
	}
	return hashes
}

func (m *MessageTxsResponse) Hashes() common.Hashes {
	var hashes common.Hashes
	if m.RawTxs == nil || len(*m.RawTxs) == 0 {
		return nil
	}
	for _, tx := range *m.RawTxs {
		if tx == nil {
			continue
		}
		hashes = append(hashes, GetTxHash())
	}
	if m.RawSequencer != nil {
		hashes = append(hashes, m.RawSequencer.GetTxHash())
	}
	return hashes
}
