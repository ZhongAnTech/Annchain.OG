package txcache

import (
	"github.com/Metabdulla/gcache"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"time"
)

type TxCache  struct {
	  cache          gcache.OrderedCache
}


func NewTxCache (maxSize int, expire int ,invalidTx func(h types.Hash) bool )*TxCache {
	expireFunction :=  func ( key  interface{}) bool {
		hash:= key.(types.Hash)
		return  invalidTx(hash)
	}
	return &TxCache{
		cache: gcache.New(maxSize).Expiration(
			time.Second * time.Duration(expire)).ExpiredFunc(expireFunction).BuildOrderedCache(),
	}
}

func (t *TxCache)GetHashOrder () types.Hashes{
	v := t.cache.OrderedKeys()
	var hashes types.Hashes
	for _ ,k := range v {
		hash := k.(types.Hash)
		hashes = append(hashes,hash)
	}
	return hashes
}

//Get get an item
func (t *TxCache) Get(h types.Hash) types.Txi {
	v,err :=  t.cache.GetIFPresent(h)
	if err == nil {
		return  v.(types.Txi)
	}
	return nil
}

func ( t *TxCache)Has(h types.Hash)  bool {
	_, err :=  t.cache.GetIFPresent(h)
	if err == nil {
		return  true
	}
	return false
}


// Add tx into txCache
func (t *TxCache) EnQueue(tx  types.Txi) error {
	start := time.Now()
	err:=  t.cache.EnQueue(tx.GetTxHash(),tx)
	log.WithField("enqueued tx",tx).WithField("used",time.Now().Sub(start)).Debug("enqueue total")
	return err
}


//get top element end remove it
func (t *TxCache)DeQueue () types.Txi {
	_, value ,err :=  t.cache.DeQueue()
	if err!=nil {
		return  nil
	}
	return value.(types.Txi)
}

// Remove tx from txCache , too slow ,don't use this
func (t *TxCache) Remove(h types.Hash) bool	 {
	return  t.cache.Remove(h)
}

func ( t*TxCache)Len()int {
	 return  t.cache.Len()
}

//
func ( t*TxCache)RemoveExpiredAndInvalid( allowFailCount int ) error  {
	log.WithField("total cache len" ,t.cache.Len()).Debug("before remove expired ")
	defer log.WithField("total cache len" ,t.cache.Len()).Debug("after remove expired ")
	return  t.cache.RemoveExpired(allowFailCount)
}

func (t *TxCache)Refresh() {
	 t.cache.Refresh()
}


func (c*TxCache)DeQueueBatch(count int) (txs types.Txis,err error) {
	_, values,err:=  c.cache.DeQueueBatch(count)
	if err!=nil {
		return nil, err
	}
	for _, v:= range values {
		txi := v.(types.Txi)
		txs = append(txs,txi)
	}
	return txs,nil
}

// Add tx into txCache
func (t *TxCache) Prepend(tx  types.Txi) error {
	start := time.Now()
	err:=  t.cache.Prepend(tx.GetTxHash(),tx)
	log.WithField("Prepend tx",tx).WithField("used",time.Now().Sub(start)).Debug("Prepend total")
	return err
}


func (c*TxCache) PrependBatchTxs( txs types.Txs) error {
	if len(txs) == 0 {
		return nil
	}
	var  keys []interface{}
	var values  []interface{}
	for _,tx := range txs {
		keys =  append(keys,tx.GetTxHash())
		txi := types.Txi(tx)
		values = append(values,txi)
	}
	return c.cache.PrependBatch(keys,values)
}

func (c*TxCache) PrependBatchTxis( txis types.Txis) error {
	if len(txis) == 0 {
		return nil
	}
	var  keys []interface{}
	var values  []interface{}
	for _,tx := range txis {
		keys =  append(keys,tx.GetTxHash())
		txi := types.Txi(tx)
		values = append(values,txi)
	}
	return c.cache.PrependBatch(keys,values)
}

func (c*TxCache) PrependBatchSeqs( seqs types.Sequencers) error {
	if len(seqs) == 0 {
		return nil
	}
	var  keys []interface{}
	var values  []interface{}
	for _,tx := range seqs {
		keys =  append(keys,tx.GetTxHash())
		txi := types.Txi(tx)
		values = append(values,txi)
	}
	return c.cache.PrependBatch(keys,values)
}

func (c*TxCache) PrependBatch( txs types.Txis) error {
	if len(txs) == 0 {
		return nil
	}
	var  keys []interface{}
	var values  []interface{}
	for _,tx := range txs {
		keys =  append(keys,tx.GetTxHash())
		values = append(values,tx)
	}
	return c.cache.PrependBatch(keys,values)
}

func (c*TxCache) EnQueueBatchTxs( txs types.Txs) error {
	if len(txs) == 0 {
		return nil
	}
     var  keys []interface{}
     var values  []interface{}
     for _,tx := range txs {
     	keys =  append(keys,tx.GetTxHash())
     	txi := types.Txi(tx)
		 values = append(values,txi)
	 }
	return c.cache.EnQueueBatch(keys,values)
}

func (c*TxCache) EnQueueBatchSeqs( seqs types.Sequencers) error {
	if len(seqs) == 0 {
		return nil
	}
	var  keys []interface{}
	var values  []interface{}
	for _,tx := range seqs {
		keys =  append(keys,tx.GetTxHash())
		txi := types.Txi(tx)
		values = append(values,txi)
	}
	return c.cache.EnQueueBatch(keys,values)
}

func (c*TxCache) EnQueueBatch( txs types.Txis) error {
	if len(txs) == 0 {
		return nil
	}
	var  keys []interface{}
	var values  []interface{}
	for _,tx := range txs {
		keys =  append(keys,tx.GetTxHash())
		values = append(values,tx)
	}
	return c.cache.EnQueueBatch(keys,values)
}

func (t*TxCache)GetTop()types.Txi{
	_, value ,err :=  t.cache.GetTop()
	if err!=nil {
		return  nil
	}
	return value.(types.Txi)
}


//MoveFront move an element to font
func (t*TxCache)MoveFront(tx types.Txi) error  {
	defer log.WithField(" tx",tx).Debug("moved to front")
	return  t.cache.MoveFront(tx.GetTxHash())
}


