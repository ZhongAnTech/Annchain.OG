package syncer

import (
	"github.com/annchain/OG/types"
)

func (m *IncrementalSyncer) HandleNewTxi(tx types.Txi) {
	// cancel pending requests if it is there
	if !m.Enabled {
		if !m.cacheNewTxEnabled() {
			log.Debug("incremental received nexTx but sync disabled")
			return
		}
	}
	m.RemoveContrlMsgFromCache(tx.GetTxHash())
	m.firedTxCache.Remove(tx.GetTxHash())
	if m.isKnownHash(tx.GetTxHash()) {
		log.WithField("tx ", tx).Debug("duplicated tx received")
		return
	}
	if !m.Enabled {
		log.WithField("tx ", tx).Debug("cache txs for future.")
	}
	err := m.bufferedIncomingTxCache.EnQueue(tx)
	if err != nil {
		log.WithError(err).Warn("add tx to cache error")
	}
	//m.notifyTxEvent <- true
	//notify channel will be  blocked if tps is high ,check first and add
}

func (m *IncrementalSyncer) HandleNewTx(newTx *types.MessageNewTx, peerId string) {
	tx := newTx.RawTx.Tx()
	if tx == nil {
		log.Debug("empty MessageNewTx")
		return
	}
	m.HandleNewTxi(tx)
	log.WithField("q", newTx).Debug("incremental received MessageNewTx")

}

func (m *IncrementalSyncer) HandleNewTxs(newTxs *types.MessageNewTxs, peerId string) {
	if newTxs.RawTxs == nil || len(*newTxs.RawTxs) == 0 {
		log.Debug("Empty MessageNewTx")
		return
	}
	var validTxs types.Txis
	if !m.Enabled {
		if !m.cacheNewTxEnabled() {
			log.Debug("incremental received nexTx but sync disabled")
			return
		}
	}
	for _, tx := range *newTxs.RawTxs {
		m.RemoveContrlMsgFromCache(tx.GetTxHash())
		m.firedTxCache.Remove(tx.GetTxHash())
		if m.isKnownHash(tx.GetTxHash()) {
			log.WithField("tx ", tx).Debug("duplicated tx received")
			continue
		}
		if !m.Enabled {
			log.WithField("tx ", tx).Debug("cache txs for future.")
		}
		validTxs = append(validTxs, tx.Tx())
	}

	err := m.bufferedIncomingTxCache.EnQueueBatch(validTxs)
	if err != nil {
		log.WithError(err).Warn("add tx to cache error")
	}
	log.WithField("q", newTxs).Debug("incremental received MessageNewTxs")
}

func (m *IncrementalSyncer) HandleNewSequencer(newSeq *types.MessageNewSequencer, peerId string) {
	seq := newSeq.RawSequencer.Sequencer()
	if seq == nil {
		log.Debug("empty NewSequence")
		return
	}
	m.HandleNewTxi(seq)
	log.WithField("q", newSeq).Debug("incremental received NewSequence")
}

func (m *IncrementalSyncer) HandleCampaign(request *types.MessageCampaign, peerId string) {
	cp := request.RawCampaign.Campaign()
	if cp == nil {
		log.Warn("got nil MessageCampaign")
		return
	}
	m.HandleNewTxi(cp)
	log.WithField("q", request).Debug("incremental received MessageCampaign")

}

func (m *IncrementalSyncer) HandleTermChange(request *types.MessageTermChange, peerId string) {
	cp := request.RawTermChange.TermChange()
	if cp == nil {
		log.Warn("got nil MessageCampaign")
		return
	}
	m.HandleNewTxi(cp)
	log.WithField("q", request).Debug("incremental received MessageTermChange")

}

func (m *IncrementalSyncer) HandleFetchByHashResponse(syncResponse *types.MessageSyncResponse, sourceId string) {
	m.bloomFilterStatus.UpdateResponse(syncResponse.RequestedId)
	if syncResponse.RawTxs == nil || len(*syncResponse.RawTxs) == 0 {
		log.Debug("empty MessageSyncResponse")
		return
	}
	//	if len(syncResponse.RawSequencers) != len(syncResponse.SequencerIndex) || len(syncResponse.RawSequencers) > 20 {
	//	log.WithField("length of sequencers", len(syncResponse.RawSequencers)).WithField(
	//	"length of index", len(syncResponse.SequencerIndex)).Warn("is it an attacking ?????")
	//return
	//}

	var txis types.Txis
	//
	//var currentIndex int
	//var testVal int
	//testVal = len(syncResponse.RawSequencers)
	//}
	if syncResponse.RawTxs != nil {
		for _, rawTx := range *syncResponse.RawTxs {
			/*for ; currentIndex < len(syncResponse.RawSequencers) &&
				uint32(i) == syncResponse.SequencerIndex[currentIndex] ;currentIndex++ {
				testVal++
				tx := syncResponse.RawSequencers[currentIndex].Sequencer()
				m.firedTxCache.Remove(tx.Hash)
				if m.isKnownHash(tx.GetTxHash()) {
					log.WithField("tx ", tx).Debug("duplicated tx received")
				} else {
					if !m.Enabled {
						log.WithField("tx ", tx).Debug("cache seqs for future.")
					}
					log.WithField("seq ", tx).Debug("received sync response seq")
					txis = append(txis, tx)
				}

			}
			*/
			if rawTx == nil {
				log.Warn("nil tx received")
				continue
			}
			m.RemoveContrlMsgFromCache(rawTx.GetTxHash())
			m.firedTxCache.Remove(rawTx.GetTxHash())
			if m.isKnownHash(rawTx.GetTxHash()) {
				log.WithField("tx ", rawTx).Debug("duplicated tx received")
				continue
			}
			if !m.Enabled {
				log.WithField("tx ", rawTx).Debug("cache txs for future.")
			}
			txis = append(txis, rawTx.Txi())
		}
	}
	//if testVal!=len(syncResponse.RawSequencers) {
	//panic(fmt.Sprintf("algorithm err ,len mismatch, %d,%d ",testVal, len(syncResponse.RawSequencers)))
	//}
	err := m.bufferedIncomingTxCache.PrependBatch(txis)
	if err != nil {
		log.WithError(err).Warn("add txs error")
	}
	//if len(txis) > 10 {
	//	start := time.Now()
	//	m.bufferedIncomingTxCache.Sort()
	//	now := time.Now()
	//	log.WithField("len ", m.bufferedIncomingTxCache.Len()).WithField("used for sort ", now.Sub(start)).Debug("sorted data")
	//}
	log.WithField("txis", txis).WithField("peer", sourceId).Debug("received sync response Txs")
}
