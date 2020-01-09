// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package syncer

import (
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/p2p_message"
	"sort"
)

func (m *IncrementalSyncer) HandleNewTxi(tx types.Txi, peerId string) {
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

	if tx.GetType() == types.TxBaseTypeSequencer {
		m.SequencerCache.Add(tx.GetTxHash(), peerId)
	}

	err := m.bufferedIncomingTxCache.EnQueue(tx)
	if err != nil {
		log.WithError(err).Warn("add tx to cache error")
	}
	if m.Enabled {
		m.notifyTxEvent <- true
	}
	//m.notifyTxEvent <- true
	//notify channel will be  blocked if tps is high ,check first and add
}

func (m *IncrementalSyncer) HandleNewTx(newTx *p2p_message.MessageNewTx, peerId string) {
	tx := newTx.RawTx.Tx()
	if tx == nil {
		log.Debug("empty MessageNewTx")
		return
	}
	m.HandleNewTxi(tx, peerId)
	log.WithField("q", newTx).Debug("incremental received MessageNewTx")

}

func (m *IncrementalSyncer) HandleNewTxs(newTxs *p2p_message.MessageNewTxs, peerId string) {
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
	if m.Enabled {
		m.notifyTxEvent <- true
	}
	log.WithField("q", newTxs).Debug("incremental received MessageNewTxs")
}

func (m *IncrementalSyncer) HandleNewSequencer(newSeq *p2p_message.MessageNewSequencer, peerId string) {
	seq := newSeq.RawSequencer.Sequencer()
	if seq == nil {
		log.Debug("empty NewSequence")
		return
	}
	m.HandleNewTxi(seq, peerId)
	log.WithField("q", newSeq).Debug("incremental received NewSequence")
}

func (m *IncrementalSyncer) HandleCampaign(request *p2p_message.MessageCampaign, peerId string) {
	cp := request.RawCampaign.Campaign()
	if cp == nil {
		log.Warn("got nil MessageCampaign")
		return
	}
	m.HandleNewTxi(cp, peerId)
	log.WithField("q", request).Debug("incremental received MessageCampaign")

}

func (m *IncrementalSyncer) HandleTermChange(request *p2p_message.MessageTermChange, peerId string) {
	cp := request.RawTermChange.TermChange()
	if cp == nil {
		log.Warn("got nil MessageCampaign")
		return
	}
	m.HandleNewTxi(cp, peerId)
	log.WithField("q", request).Debug("incremental received MessageTermChange")

}

func (m *IncrementalSyncer) HandleArchive(request *p2p_message.MessageNewArchive, peerId string) {
	ac := request.Archive
	if ac == nil {
		log.Warn("got nil MessageNewArchive")
		return
	}
	m.HandleNewTxi(ac, peerId)
	log.WithField("q", request).Debug("incremental received MessageNewArchive")

}

func (m *IncrementalSyncer) HandleActionTx(request *p2p_message.MessageNewActionTx, peerId string) {
	ax := request.ActionTx
	if ax == nil {
		log.Warn("got nil MessageNewActionTx")
		return
	}
	m.HandleNewTxi(ax, peerId)
	log.WithField("q", request).Debug("incremental received MessageNewActionTx")
}

func (m *IncrementalSyncer) HandleFetchByHashResponse(syncResponse *p2p_message.MessageSyncResponse, sourceId string) {
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
	//if testVal!=len(syncResponse.RawSequencers) {
	//panic(fmt.Sprintf("algorithm err ,len mismatch, %d,%d ",testVal, len(syncResponse.RawSequencers)))
	//}
	sort.Sort(txis)
	err := m.bufferedIncomingTxCache.PrependBatch(txis)
	if err != nil {
		log.WithError(err).Warn("add txs error")
	}
	if m.Enabled {
		m.notifyTxEvent <- true
	}
	//if len(txis) > 10 {
	//	start := time.Now()
	//	m.bufferedIncomingTxCache.Sort()
	//	now := time.Now()
	//	log.WithField("len ", m.bufferedIncomingTxCache.Len()).WithField("used for sort ", now.Sub(start)).Debug("sorted data")
	//}
	log.WithField("txis", txis).WithField("peer", sourceId).Debug("received sync response Txs")
}
