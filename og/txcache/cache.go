package txcache

import (
	"errors"
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

var ReachedMaxSizeErr = errors.New("reachedMaxSize")
var DuplicateErr  = errors.New("duplicate")
type TxCache  struct {
	  cache          gcache.Cache
	  order          types.Hashes
	  mu     	     sync.RWMutex
	  maxSize           int
	  invalidTx         func (h types.Hash) bool
}


func NewTxCache (maxSize int, expire int ,invalidTx func(h types.Hash) bool )*TxCache {
	return &TxCache{
		cache: gcache.New(maxSize).Simple().
			Expiration(time.Second * time.Duration(expire)).Build(),
		maxSize:maxSize,
		invalidTx:invalidTx,
	}
}

func (t *TxCache)GetHashOrder () types.Hashes{
	return t.order
}

func (t * TxCache)MaxSize()int {
	return t.maxSize
}

func (t *TxCache) get(h types.Hash) types.Txi {
	v,err :=  t.cache.GetIFPresent(h)
	if err == nil {
		return  v.(types.Txi)
	}
	return nil
}

//Get get an item
func(t *TxCache)Get(h types.Hash) types.Txi {
	return  t.get(h)
}

func ( t *TxCache)Has(h types.Hash)  bool {
	return  t.has(h)
}

func (t*TxCache)has(h types.Hash) bool {
	_, err :=  t.cache.GetIFPresent(h)
	if err == nil {
		return  true
	}
	return false
}


// Add tx into txCache
func (t *TxCache) EnQueue(tx  types.Txi) error {
	t.mu.Lock()
	defer  t.mu.Unlock()
	start := time.Now()
	err :=t.set(tx)
	//hash := tx.GetTxHash()
	now:=time.Now()
	log.WithField("used",now.Sub(start)).Debug(" enqueue after set")
	if err==nil {

		//t.order = append(t.order,hash)

		now :=time.Now()
		log.WithField("used",now.Sub(start)).Debug("enqueue after append")
		//
	}
	//log.WithField("used",time.Now().Sub(start)).Debug("enqueue total")
	return err
}

func (t*TxCache)set(tx types.Txi) error {
	if t.has(tx.GetTxHash()) {
		return DuplicateErr
	}
	if  len(t.order)  >= t.maxSize   {
		return  ReachedMaxSizeErr
	}
	err:= t.cache.Set(tx.GetTxHash(),tx)
	if err!= nil {
		log.WithError(err).Error("add cache error")
		return err
	}
	t.order = append(t.order,tx.GetTxHash())
	return nil
}

//get top element end remove it
func (t *TxCache)DeQueue () types.Txi {
	t.mu.Lock()
	defer  t.mu.Unlock()
	return t.deQueue()
}


func (t *TxCache)deQueue() types.Txi {
	var tx types.Txi
	for len(t.order)!=0  {
		hash := t.order[0]
		t.order = t.order[1:]
		tx = t.get(hash)
		if tx!=nil{
			break
		}
	}
	if tx!=nil {
		t.cache.Remove(tx)
	}
	return nil
}

func (t*TxCache)Empty()bool {
	t.mu.Lock()
	defer  t.mu.Unlock()
	return len(t.order) == 0
}



// Remove tx from txCache , too slow ,don't use this
func (t *TxCache) Remove(h types.Hash) {
	t.mu.Lock()
	defer  t.mu.Unlock()
	t.remove(h)
}

func ( t*TxCache)Len()int {
	if t==nil {
		return 0
	}
	t.mu.Lock()
	defer  t.mu.Unlock()
	return  len(t.order)
}

//get all tx in order and remove them from cache
func ( t*TxCache)PopALl() types.Txis{
	t.mu.Lock()
	defer  t.mu.Unlock()
	return t.popALl()
}

func ( t*TxCache)popALl () types.Txis{
	var txs types.Txis
	hashes := t.order
	for _, hash := range hashes {
		tx := t.get(hash)
		if tx!=nil {
			txs = append(txs,tx)
			t.cache.Remove(hash)
		}
	}
	t.order = nil
	return txs
}

func (t*TxCache)remove( h types.Hash) bool {
	ok := t.cache.Remove(h)
	for i, hash := range t.order {
		if hash.Cmp(h) == 0 {
			t.order = append(t.order[:i], t.order[i+1:]...)
		}
	}
	return ok
}

//
func ( t*TxCache)RemoveExpiredAndInvalid( isInValidTx  func(h types.Hash)bool, full bool) bool  {
	t.mu.Lock()
	defer  t.mu.Unlock()
	return t.removeExpiredAndInvalid(isInValidTx, full)
}

func (t*TxCache)removeExpiredAndInvalid(isInValidTx  func(h types.Hash) bool,full bool) bool  {
	var removed bool
	for i:=0;  i< len(t.order);  {
		hash := t.order[i]
		tx := t.get(hash)
		if tx!=nil {
			if isInValidTx(hash) {
				t.cache.Remove(hash)
				log.WithField("hash",hash).Debug("remove expired tx")
			}else {
				if !full{
					return  removed
				}
				i++
				continue
			}
		}
		log.WithField("hash",hash).Debug("remove expired hash")
		t.order = append(t.order[:i], t.order[i+1:]...)
		removed = true
	}
	return removed
}

func (t*TxCache)GetTop()types.Txi{
	t.mu.Lock()
	defer  t.mu.Unlock()
	return  t.getTop()
}

func (t *TxCache)getTop() types.Txi {
	for len(t.order)!=0  {
		h:= t.order[0]
		tx := t.get(h)
		if tx!=nil {
			return tx
		}
		t.order = t.order[1:]
	}
	return nil
}

func (t*TxCache)TryDeQueue(h types.Hash) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return  t.tryDeQueue(h)
}

func (t*TxCache)tryDeQueue(h types.Hash) bool {
		ok :=t.cache.Remove(h)
		if ok {
			for t.order.Len() != 0 {
				if t.order[0] == h {
					t.order = t.order[1:]
					return true
				} else {
					//some hash moved to front, do nothing
					if t.get(t.order[0]) != nil {
						return true
					} else {
						//expired hash
						t.order = t.order[1:]
					}
				}
			}
			return true
		}

	return false
}

//MoveFront move an element to font
func (t*TxCache)MoveFront(h types.Hash) bool  {
	t.mu.Lock()
	defer  t.mu.Unlock()
	return  t.moveFront(h)
}


//moveFront  move an element to front

func (t*TxCache)moveFront(h types.Hash)  bool  {
	tx := t.get(h)
	if tx!=nil {
		var hashes types.Hashes
		hashes = append(hashes,h)
		hashes = append(hashes,t.order...)
		t.order = hashes
		return  true
	}
	return false
}