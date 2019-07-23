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
package txcache

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/gcache"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"sort"
	"time"
)

type TxCache struct {
	cache gcache.OrderedCache
}

func newCacheItemCmpFunction() gcache.SearchCompareFunction {
	cmpFunc := func(value interface{}, anotherValue interface{}) int {
		tx1 := value.(types.Txi)
		tx2 := anotherValue.(types.Txi)
		if tx1.GetWeight() > tx2.GetWeight() {
			return 1
		} else if tx1.GetWeight() < tx2.GetWeight() {
			return -1
		}
		return 0
	}
	return cmpFunc
}

func newCacheItemSortFunction() gcache.SortKeysFunction {
	var allItem = false
	sortByAllValue := func(keys []interface{}, values []interface{}, getItem func(key interface{}) (interface{}, bool)) (sortedKeys []interface{}, sortOk bool) {
		var txis types.Txis
		for i, val := range values {
			tx := val.(types.Txi)
			if tx == nil {
				log.WithField("i", i).Error("got nil tx")
				continue
			}
			txis = append(txis, tx)
		}
		sort.Sort(txis)
		for _, tx := range txis {
			sortedKeys = append(sortedKeys, tx.GetTxHash())
		}
		if len(keys) != len(sortedKeys) {
			log.WithField("len keys", len(keys)).WithField("len txis ", len(txis)).Info("sorted tx")
		}
		return sortedKeys, true
	}
	sortByGetEachItem := func(keys []interface{}, values []interface{}, getItem func(key interface{}) (interface{}, bool)) (sortedKeys []interface{}, sortOk bool) {
		var txis types.Txis
		for i, k := range keys {
			val, ok := getItem(k)
			if !ok {
				continue
			}
			tx := val.(types.Txi)
			if tx == nil {
				log.WithField("i", i).WithField("key ", k).Error("got nil tx")
				continue
			}
			txis = append(txis, tx)
		}
		sort.Sort(txis)
		for _, tx := range txis {
			sortedKeys = append(sortedKeys, tx.GetTxHash())
		}
		if len(keys) != len(sortedKeys) {
			log.WithField("len keys", len(keys)).WithField("len txis ", len(txis)).Info("sorted tx")
		}
		return sortedKeys, true
	}
	if allItem {
		return sortByAllValue
	}
	return sortByGetEachItem
}

func NewTxCache(maxSize int, expire int, invalidTx func(h common.Hash) bool, sorted bool) *TxCache {

	expireFunction := func(key interface{}) bool {
		hash := key.(common.Hash)
		return invalidTx(hash)
	}
	sortFunc := newCacheItemSortFunction()
	var cmpFunction gcache.SearchCompareFunction
	if sorted {
		cmpFunction = newCacheItemCmpFunction()
	} else {
		cmpFunction = nil
	}
	return &TxCache{
		cache: gcache.New(maxSize).Expiration(
			time.Second * time.Duration(expire)).ExpiredFunc(expireFunction).SortKeysFunc(sortFunc).SearchCompareFunction(
			cmpFunction).BuildOrderedCache(),
	}

}

func (t *TxCache) GetHashOrder() common.Hashes {
	v := t.cache.OrderedKeys()
	var hashes common.Hashes
	for _, k := range v {
		hash := k.(common.Hash)
		hashes = append(hashes, hash)
	}
	return hashes
}

//Get get an item
func (t *TxCache) Get(h common.Hash) types.Txi {
	v, err := t.cache.GetIFPresent(h)
	if err == nil {
		return v.(types.Txi)
	}
	return nil
}

func (t *TxCache) Has(h common.Hash) bool {
	_, err := t.cache.GetIFPresent(h)
	if err == nil {
		return true
	}
	return false
}

// Add tx into txCache
func (t *TxCache) EnQueue(tx types.Txi) error {
	err := t.cache.EnQueue(tx.GetTxHash(), tx)
	//log.WithField("enqueued tx",tx).WithField("used",time.Now().Sub(start)).Debug("enqueue total")
	return err
}

//get top element end remove it
func (t *TxCache) DeQueue() types.Txi {
	_, value, err := t.cache.DeQueue()
	if err != nil {
		return nil
	}
	return value.(types.Txi)
}

// Remove tx from txCache
func (t *TxCache) Remove(h common.Hash) bool {
	return t.cache.Remove(h)
}

func (t *TxCache) Len() int {
	return t.cache.Len()
}

//
func (t *TxCache) RemoveExpiredAndInvalid(allowFailCount int) error {
	return t.cache.RemoveExpired(allowFailCount)
}

func (t *TxCache) Refresh() {
	t.cache.Refresh()
}

func (c *TxCache) DeQueueBatch(count int) (txs types.Txis, err error) {
	_, values, err := c.cache.DeQueueBatch(count)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		txi := v.(types.Txi)
		txs = append(txs, txi)
	}
	return txs, nil
}

// Add tx into txCache
func (t *TxCache) Prepend(tx types.Txi) error {
	start := time.Now()
	err := t.cache.Prepend(tx.GetTxHash(), tx)
	log.WithField("Prepend tx", tx).WithField("used", time.Now().Sub(start)).Debug("Prepend total")
	return err
}

func (c *TxCache) PrependBatch(txs types.Txis) error {
	if len(txs) == 0 {
		return nil
	}
	sort.Sort(txs)
	var keys []interface{}
	var values []interface{}
	for _, tx := range txs {
		keys = append(keys, tx.GetTxHash())
		values = append(values, tx)
	}
	start := time.Now()
	log.WithField("len ", len(keys)).Debug("before prepend keys")
	err := c.cache.PrependBatch(keys, values)
	log.WithField("used time ", time.Now().Sub(start)).WithField("len ", len(keys)).Debug("after prepend keys")
	return err
}

func (c *TxCache) EnQueueBatch(txs types.Txis) error {
	if len(txs) == 0 {
		return nil
	}
	sort.Sort(txs)
	var keys []interface{}
	var values []interface{}
	for _, tx := range txs {
		keys = append(keys, tx.GetTxHash())
		values = append(values, tx)
	}
	return c.cache.EnQueueBatch(keys, values)
}

func (t *TxCache) GetTop() types.Txi {
	_, value, err := t.cache.GetTop()
	if err != nil {
		return nil
	}
	return value.(types.Txi)
}

//MoveFront move an element to font, if searchFunction is set
//func (t *TxCache) MoveFront(tx types.Txi) error {
//	defer log.WithField(" tx", tx).Debug("moved to front")
//	return t.cache.MoveFront(tx.GetTxHash())
//}

func (t *TxCache) Sort() {
	t.cache.Sort()
}
