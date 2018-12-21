package syncer

import (
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
	"sync"
	"time"
)

type TxCache  struct {
	  cache            gcache.Cache
	  order          types.Hashes
	  mu     sync.RWMutex
}

func newTxCache (size int, expire int )*TxCache {
	return &TxCache{
		cache: gcache.New(size).Simple().
			Expiration(time.Second * time.Duration(expire)).Build(),
	}
}

func (t *TxCache) get(h types.Hash) types.Txi {
	v,err :=  t.cache.GetIFPresent(h)
	if err == nil {
		return  v.(types.Txi)
	}
	return nil
}

func(t *TxCache)Get(h types.Hash) types.Txi {
	return t.get(h)
}

func ( t *TxCache)Has(h types.Hash)  bool {
	_, err :=  t.cache.GetIFPresent(h)
	if err == nil {
		return  true
	}
	return false
}

// Add tx into txcache
func (t *TxCache) Set(tx  types.Txi) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return  t.set(tx)
}

func (t *TxCache) set(tx types.Txi) bool  {
	if t.Has(tx.GetTxHash()) {
		return false
	}
	err:= t.cache.Set(tx.GetTxHash(),tx)
	if err!= nil {
		log.WithError(err).Error("add cache error")
		return false
	}
	t.order = append(t.order, tx.GetTxHash())
    return true
}

// Remove tx from txLookUp
func (t *TxCache) Remove(h types.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.remove(h)
}
func (t *TxCache) remove(h types.Hash)  bool {
	for i, hash := range t.order {
		if hash.Cmp(h) == 0 {
			t.order = append(t.order[:i], t.order[i+1:]...)
		}
	}
	return t.cache.Remove(h)
}

// RemoveByIndex removes a tx by its order index
func (t *TxCache) RemoveByIndex(i int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.removeByIndex(i)
}
