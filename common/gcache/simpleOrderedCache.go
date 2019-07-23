package gcache

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

// SimpleOrderedCache has no clear priority for evict cache. It depends on key-value map order.
type SimpleOrderedCache struct {
	baseCache
	items       map[interface{}]*simpleItem
	orderedKeys []interface{}
}

func newSimpleOrderedCache(cb *CacheBuilder) *SimpleOrderedCache {
	c := &SimpleOrderedCache{}
	buildCache(&c.baseCache, cb)

	c.init()
	c.loadGroup.orderedCache = c
	return c
}

func (c *SimpleOrderedCache) init() {
	if c.size <= 0 {
		c.items = make(map[interface{}]*simpleItem)
	} else {
		c.items = make(map[interface{}]*simpleItem, c.size)
	}
	c.orderedKeys = nil
}

func (c *SimpleOrderedCache) addFront(key, value interface{}) (interface{}, error) {
	var err error
	if c.serializeFunc != nil {
		value, err = c.serializeFunc(key, value)
		if err != nil {
			return nil, err
		}
	}

	// Check for existing item
	item, ok := c.items[key]
	if ok {
		if c.searchCmpFunc == nil {
			item.value = value
		} else {
			//don't set again ,if using ordered insert
			return item, fmt.Errorf("duplicated set")
		}
	} else {
		// Verify size not exceeded
		if (len(c.items) >= c.size) && c.size > 0 {
			c.evict(1)
			if len(c.items) >= c.size {
				return nil, ReachedMaxSizeErr
			}
		}
		item = &simpleItem{
			clock: c.clock,
			value: value,
		}
		c.items[key] = item
		if c.searchCmpFunc != nil {
			c.insertKey(key, value, 0)
		} else {
			c.orderedKeys = append([]interface{}{key}, c.orderedKeys...)
		}
	}

	if c.expiration != nil {
		t := c.clock.Now().Add(*c.expiration)
		item.expiration = &t
	}

	if c.addedFunc != nil {
		c.addedFunc(key, value)
	}

	return item, nil
}

func (c *SimpleOrderedCache) enQueue(key, value interface{}) (interface{}, error) {
	var err error
	if c.serializeFunc != nil {
		value, err = c.serializeFunc(key, value)
		if err != nil {
			return nil, err
		}
	}

	// Check for existing item
	item, ok := c.items[key]
	if ok {
		if c.searchCmpFunc == nil {
			item.value = value
		} else {
			//don't set again ,if using ordered insert
			return item, fmt.Errorf("duplicated set")
		}

	} else {
		// Verify size not exceeded
		if (len(c.items) >= c.size) && c.size > 0 {
			c.evict(1)
			if len(c.items) >= c.size {
				return nil, ReachedMaxSizeErr
			}
		}
		item = &simpleItem{
			clock: c.clock,
			value: value,
		}
		c.items[key] = item
		if c.searchCmpFunc != nil {
			c.insertKey(key, value, len(c.orderedKeys))
		} else {
			c.orderedKeys = append(c.orderedKeys, key)
		}
	}

	if c.expiration != nil {
		t := c.clock.Now().Add(*c.expiration)
		item.expiration = &t
	}

	if c.addedFunc != nil {
		c.addedFunc(key, value)
	}

	return item, nil
}

func (c *SimpleOrderedCache) enQueueBatch(keys []interface{}, values []interface{}) error {
	if len(values) != len(keys) {
		panic("len(keys) != len(values)")
	}
	evicted := false
	var insertKeys []interface{}
	var insertValues []interface{}
	var err error
	for i, key := range keys {
		value := values[i]
		if c.serializeFunc != nil {
			value, err = c.serializeFunc(key, value)
			if err != nil {
				break
			}
		}

		// Check for existing item
		item, ok := c.items[key]
		if ok {
			if c.searchCmpFunc == nil {
				item.value = value
			} else {
				//don't set again ,if using ordered insert
			}
		} else {
			// Verify size not exceeded
			if (len(c.items) >= c.size) && c.size > 0 {
				if !evicted {
					c.evict(len(keys) - i)
					evicted = true
				}
				if len(c.items) >= c.size {
					err = ReachedMaxSizeErr
				}
			}
			item = &simpleItem{
				clock: c.clock,
				value: value,
			}
			c.items[key] = item
			if c.searchCmpFunc != nil {
				insertKeys = append(insertKeys, key)
				insertValues = append(insertValues, value)
			} else {
				c.orderedKeys = append(c.orderedKeys, key)
			}
		}

		if c.expiration != nil {
			t := c.clock.Now().Add(*c.expiration)
			item.expiration = &t
		}

		if c.addedFunc != nil {
			c.addedFunc(key, value)
		}
	}
	if c.searchCmpFunc != nil {
		c.insertKeys(insertKeys, insertValues, len(c.orderedKeys))
	}
	return err
}

//insertKeys  required  keys and values are sorted , otherwise never call this function
func (c *SimpleOrderedCache) insertKeys(keys []interface{}, values []interface{}, suggestedAt int) {
	if c.searchCmpFunc == nil {
		panic("index func is nil")
	}
	if len(keys) == 0 {
		return
	}
	if suggestedAt < 0 || suggestedAt > len(c.orderedKeys) || len(keys) != len(values) {
		panic(fmt.Sprintf("suggestedAt %d out of range ,len: %d, len keys %d, len values:%d",
			suggestedAt, len(c.orderedKeys), len(keys), len(values)))
	}
	if len(c.orderedKeys) == 0 {
		log.WithField("keys ", keys).Trace("ordered keys is nil ")
		c.orderedKeys = keys
		return
	}
	if len(keys) == 1 {
		c.insertKey(keys[0], values[0], suggestedAt)
		return
	}
	// if all item are smaller than first cache item, insert all of them to front
	first := 0
	ok := false
	for !ok {
		k := c.orderedKeys[first]
		item, ok := c.items[k]
		if ok {
			if c.searchCmpFunc(values[len(values)-1], item.value) <= 0 {
				newKeys := keys
				c.orderedKeys = append(newKeys, c.orderedKeys[first:]...)
				log.WithField("keys", keys).WithField("orderedkeys ", len(c.orderedKeys)).Trace("append keys to front")
				return
			}
			break
		}
		//may be expired
		first++
		if first == len(c.orderedKeys) {
			c.orderedKeys = keys
			log.WithField("keys", keys).WithField("orderedkeys ", len(c.orderedKeys)).Trace("append keys to front, end")
			return
		}
	}
	//if all item are bigger than last item ,insert all of them to end
	last := len(c.orderedKeys) - 1
	ok = false
	for !ok {
		k := c.orderedKeys[last]
		item, ok := c.items[k]
		if ok {
			if c.searchCmpFunc(values[0], item.value) >= 0 {
				if last != len(c.orderedKeys)-1 {
					c.orderedKeys = append(c.orderedKeys[:last+1])
				}
				c.orderedKeys = append(c.orderedKeys, keys...)
				log.WithField("keys", keys).WithField("orderedkeys ", len(c.orderedKeys)).Trace("append keys to last")
				return
			}
			break
		}
		last--
		if last < 0 {
			c.orderedKeys = keys
			log.WithField("keys", keys).WithField("orderedkeys ", len(c.orderedKeys)).Trace("append keys to last")
			return
		}
	}

	//batch search a small sorted slice from a large sorted slice
	to := len(c.orderedKeys)
	from := 0
	insertIndex := make([]int, len(keys))
	i := 0
	j := len(keys) - 1
	for i <= j {
		smallValue := values[i]
		smallIndex := c.searchFrom(to, from, smallValue)
		insertIndex[i] = smallIndex
		if i==j {
			break
		}
		from = smallIndex
		bigValue := values[j]
		if c.searchCmpFunc(smallValue, bigValue) == 0 {
			for i <= j {
				insertIndex[j] = smallIndex
				insertIndex[i] = smallIndex
				j--
				i++
			}
		}
		bigIndex := c.searchFrom(to, from, bigValue)
		insertIndex[j] = bigIndex
		if bigIndex < len(c.orderedKeys) {
			to = bigIndex + 1
		}
		j--
		i++
	}
	//then insert the values
	c.insertKeysByIndex(insertIndex, keys)
	log.WithField("inset index ", insertIndex).WithField("order ", c.orderedKeys).WithField("keys ", keys).WithField("values ",values).Trace("will insert")
	log.WithFields(log.Fields{"key": keys, "values": values, "ordered keys ": len(c.orderedKeys), "suggestedAt": suggestedAt}).Trace("after insert keys")
	return
}

func (c *SimpleOrderedCache) insertKey(key interface{}, value interface{}, suggestedAt int) {
	if c.searchCmpFunc == nil {
		panic("searchCmp function func is nil")
	}
	if len(c.orderedKeys) == 0 {
		c.orderedKeys = append(c.orderedKeys, key)
		log.WithField("key ", key).WithField("ordered ", len(c.orderedKeys)).Trace("ordered keys is nil")
		return
	}
	if key == nil {
		return
	}
	if suggestedAt < 0 || suggestedAt > len(c.orderedKeys) {
		panic(fmt.Sprintf("suggestedAt %d out of range ,len: %d", suggestedAt, len(c.orderedKeys)))
	}
	//suggested index just support front and end
	if suggestedAt == len(c.orderedKeys) {
		suggestedAt--
		k := c.orderedKeys[suggestedAt]
		item, ok := c.items[k]
		if ok {
			val := item.value
			//suggested at insert front
			if c.searchCmpFunc(value, val) >= 0 {
				c.orderedKeys = append(c.orderedKeys, key)
				log.WithField("key ", key).WithField("ordered ", len(c.orderedKeys)).Trace("insert to end")
				return
			}
		}
	} else if suggestedAt == 0 {
		k := c.orderedKeys[suggestedAt]
		item, ok := c.items[k]
		if ok {
			val := item.value
			//suggested at insert front
			if c.searchCmpFunc(value, val) <= 0 {
				c.orderedKeys = append([]interface{}{key}, c.orderedKeys...)
				log.WithField("key ", key).WithField("ordered ", len(c.orderedKeys)).Trace("insert to front")
				return
			}
		}
	} else {
		//todo
	}
	i := c.search(value)
	c.orderedKeys = SliceInsert(c.orderedKeys, i, key)
	log.WithFields(log.Fields{"key": key, "values": value, " len ordered keys ": len(c.orderedKeys), "suggestedAt": suggestedAt}).Trace("before insert key")
	return
}

func (c *SimpleOrderedCache) addFrontBatch(keys []interface{}, values []interface{}) error {
	if len(values) != len(keys) {
		panic("len(keys) != len(values)")
	}
	evicted := false
	var insertKeys []interface{}
	var insertValues []interface{}
	var err error
	for i, key := range keys {
		value := values[i]
		if c.serializeFunc != nil {
			value, err = c.serializeFunc(key, value)
			if err != nil {
				break
			}
		}

		// Check for existing item
		item, ok := c.items[key]
		if ok {
			if c.searchCmpFunc == nil {
				item.value = value
			} else {
				//don't set again ,if using ordered insert
			}
		} else {
			// Verify size not exceeded
			if (len(c.items) >= c.size) && c.size > 0 {
				if !evicted {
					c.evict(len(keys) - i)
					evicted = true
				}
				if len(c.items) >= c.size {
					err = ReachedMaxSizeErr
					break
				}
			}
			item = &simpleItem{
				clock: c.clock,
				value: value,
			}
			c.items[key] = item
			insertKeys = append(insertKeys, key)
			if c.searchCmpFunc != nil {
				insertValues = append(insertValues, value)
			}

		}

		if c.expiration != nil {
			t := c.clock.Now().Add(*c.expiration)
			item.expiration = &t
		}

		if c.addedFunc != nil {
			c.addedFunc(key, value)
		}
	}

	if c.searchCmpFunc != nil {
		c.insertKeys(insertKeys, insertValues, 0)
	} else {
		c.orderedKeys = append(insertKeys, c.orderedKeys...)
	}
	return err
}

// Get a value from cache pool using key if it exists.
// If it dose not exists key and has LoaderFunc,
// generate a value using `LoaderFunc` method returns value.
func (c *SimpleOrderedCache) Get(key interface{}) (interface{}, error) {
	v, err := c.get(key, false)
	if err == KeyNotFoundError {
		return c.getWithLoader(key, true)
	}
	return v, err
}

// Get a value from cache pool using key if it exists.
// If it dose not exists key, returns KeyNotFoundError.
// And send a request which refresh value for specified key if cache object has LoaderFunc.
func (c *SimpleOrderedCache) GetIFPresent(key interface{}) (interface{}, error) {
	v, err := c.get(key, false)
	if err == KeyNotFoundError {
		return c.getWithLoader(key, false)
	}
	return v, nil
}

func (c *SimpleOrderedCache) get(key interface{}, onLoad bool) (interface{}, error) {
	v, err := c.getValue(key, onLoad)
	if err != nil {
		return nil, err
	}
	if c.deserializeFunc != nil {
		return c.deserializeFunc(key, v)
	}
	return v, nil
}

func (c *SimpleOrderedCache) getValue(key interface{}, onLoad bool) (interface{}, error) {
	c.mu.Lock()
	item, ok := c.items[key]
	if ok {
		if !item.IsExpired(nil) {
			v := item.value
			c.mu.Unlock()
			if !onLoad {
				c.stats.IncrHitCount()
			}
			return v, nil
		}
		//todo remove ordered key
		c.delete(key)
	}
	c.mu.Unlock()
	if !onLoad {
		c.stats.IncrMissCount()
	}
	return nil, KeyNotFoundError
}

func (c *SimpleOrderedCache) getWithLoader(key interface{}, isWait bool) (interface{}, error) {
	if c.loaderExpireFunc == nil {
		return nil, KeyNotFoundError
	}
	value, _, err := c.load(key, func(v interface{}, expiration *time.Duration, e error) (interface{}, error) {
		if e != nil {
			return nil, e
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		item, err := c.enQueue(key, v)
		if err != nil {
			return nil, err
		}
		if expiration != nil {
			t := c.clock.Now().Add(*expiration)
			item.(*simpleItem).expiration = &t
		}
		return v, nil
	}, isWait)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (c *SimpleOrderedCache) evict(count int) {
	now := c.clock.Now()
	current := 0
	var fail int
	var removedKeys []int
	for i := 0; i < len(c.orderedKeys); i++ {
		if current >= count || fail > 3*count {
			break
		}
		key := c.orderedKeys[i]
		item, ok := c.items[key]
		if ok {
			if item.expiration == nil || now.After(*item.expiration) ||
				(c.expireFunction != nil && c.expireFunction(key)) {
				defer c.deleteVal(key)
				removedKeys = append(removedKeys, i)
				current++
			} else {
				fail++
			}
		} else {
			removedKeys = append(removedKeys, i)
			current++
		}
	}
	c.removeKeysByIndex(removedKeys)
}

//insert a small slice into a big slice with order
func (c *SimpleOrderedCache) insertKeysByIndex(indexes []int, keys []interface{}) {
	if len(indexes) == 0 {
		return
	}
	if len(indexes) != len(keys) {
		panic("length mismatch")
	}
	if len(c.orderedKeys) == 0 {
		c.orderedKeys = keys
		return
	}
	var result []interface{}
	first := 0
	for i, second := range indexes {
		if first > second {
			panic(fmt.Sprintf("must be sorted %d %d %v", first, second, indexes))
		}
		if second >= len(c.orderedKeys) {
			result = append(result, c.orderedKeys[first:]...)
			c.orderedKeys = append(result, keys[i:]...)
			return
		}
		if first == second {
			result = append(result, keys[i])
		} else {
			result = append(result, c.orderedKeys[first:second]...)
			result = append(result, keys[i])
		}
		first = second
	}
	result = append(result, c.orderedKeys[first:]...)
	c.orderedKeys = result
	return
}

//removeKeysByIndex
//[][][][][remove][remove][]
func (c *SimpleOrderedCache) removeKeysByIndex(indexes []int) {
	if len(indexes) == 0 {
		return
	}
	if len(indexes) > len(c.orderedKeys) {
		panic("all elements will be removed")
	}
	oldKeys := c.orderedKeys
	start := indexes[0]
	end := indexes[len(indexes)-1]
	c.orderedKeys = oldKeys[:start]
	first := start
	for i, second := range indexes {
		//just process [][][][][remove][][remove][]
		if i == 0 {
			continue
		}
		if first >= second {
			panic("must be sorted")
		}
		//don't append in this case  [][][][][j][k][]
		if first+1 < second {
			c.orderedKeys = append(c.orderedKeys, oldKeys[first+1:second]...)
		}
		first = second
	}
	c.orderedKeys = append(c.orderedKeys, oldKeys[end+1:]...)
	log.WithFields(log.Fields{"indexes": indexes, "ordered keys ": len(c.orderedKeys)}).Trace("after remove indexes")
}

func (c *SimpleOrderedCache) search(value interface{}) int {
	f := func(i int) bool {
		k := c.orderedKeys[i]
		item, ok1 := c.items[k]
		if ok1 {
			if c.searchCmpFunc(item.value, value) >= 0 {
				return true
			}
			return false
		}
		return false
	}
	return Search(len(c.orderedKeys), f)
}

func (c *SimpleOrderedCache) searchFrom(to, from int, value interface{}) int {
	f := func(i int) bool {
		k := c.orderedKeys[i]
		item, ok1 := c.items[k]
		if ok1 {
			if c.searchCmpFunc(item.value, value) >= 0 {
				return true
			}
			return false
		}
		return false
	}
	return SearchFrom(to, from, f)
}

// Removes the provided key from the cache.
func (c *SimpleOrderedCache) Remove(key interface{}) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.delete(key)
}

func (c *SimpleOrderedCache) delete(key interface{}) bool {
	log.Tracef("item will be deleted %v", key)
	item, ok := c.items[key]
	if ok {
		if c.searchCmpFunc != nil {
			var found bool
			i := c.search(item.value)
			j := i
			for ; j < len(c.orderedKeys); j++ {
				if c.orderedKeys[j] == key {
					log.Debugf("found key at index i %d , the key[i] %v ,j %d,,key [j] %v  ", i, c.orderedKeys[i], j, c.orderedKeys[j])
					c.orderedKeys = append(c.orderedKeys[:j], c.orderedKeys[j+1:]...)
					found = true
					break
				}
				itemJ, ok := c.items[c.orderedKeys[j]]
				if ok {
					if c.searchCmpFunc(itemJ.value, item.value) > 0 {
						break
					}
				}
			}
			if !found {
				if i >= len(c.orderedKeys) {
					log.Debugf("not found key i %d , j %d", i, j)
				} else if j >= len(c.orderedKeys) {
					log.Debugf("not found key i %d, j %d, key[i] %v", i, j, c.orderedKeys[i])
				} else {
					log.Debugf(" not found key at index i %d , the key[i] %v ,j %d,,key [j] %v  ", i, c.orderedKeys[i], j, c.orderedKeys[j])
				}
			}

		} else {
			j := 0
			for i, k := range c.orderedKeys {
				if k == key {
					c.orderedKeys = append(c.orderedKeys[:i], c.orderedKeys[i+1:]...)
					break
				}
				j = i
			}
			log.Debugf("cmp times %d ", j)
		}
	}
	ok = c.deleteVal(key)
	if len(c.items) == 0 {
		c.orderedKeys = nil
	} else if len(c.items) == 1 {
		c.orderedKeys = nil
		for k := range c.items {
			c.orderedKeys = append(c.orderedKeys, k)
		}
	}
	return ok
}

func (c *SimpleOrderedCache) deleteVal(key interface{}) bool {
	item, ok := c.items[key]
	if ok {
		delete(c.items, key)
		if c.evictedFunc != nil {
			c.evictedFunc(key, item.value)
		}
		return true
	}
	return false
}

// Returns a slice of the keys in the cache.
func (c *SimpleOrderedCache) keys() []interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]interface{}, len(c.items))
	var i = 0
	for k := range c.items {
		keys[i] = k
		i++
	}
	return keys
}

// Returns a slice of the keys in the cache.
func (c *SimpleOrderedCache) Keys() []interface{} {
	keys := []interface{}{}
	for _, k := range c.keys() {
		_, err := c.GetIFPresent(k)
		if err == nil {
			keys = append(keys, k)
		}
	}
	return keys
}

// Returns all key-value pairs in the cache.
func (c *SimpleOrderedCache) GetALL() map[interface{}]interface{} {
	m := make(map[interface{}]interface{})
	for _, k := range c.keys() {
		v, err := c.GetIFPresent(k)
		if err == nil {
			m[k] = v
		}
	}
	return m
}

func (c *SimpleOrderedCache) GetKeysAndValues() (keys []interface{}, values []interface{}) {
	return c.getALl()
}

func (c *SimpleOrderedCache) Refresh() {
	c.getALl()
}

func (c *SimpleOrderedCache) len() (keyLen, itemLen int) {
	c.mu.Lock()
	keyLen = len(c.orderedKeys)
	itemLen = len(c.items)
	c.mu.Unlock()
	if itemLen > keyLen {
		panic(fmt.Sprintf("item len must be smaller itemlen %d, keyLen %d", itemLen, keyLen))
	}
	return keyLen, itemLen
}

// Returns the number of items in the cache.
func (c *SimpleOrderedCache) Len() int {
	//return len(c.GetALL())
	keyLen, itemLen := c.len()
	if itemLen < keyLen/2 {
		c.getALl()
	}
	keyLen, itemLen = c.len()
	return keyLen
}

// Completely clear the cache
func (c *SimpleOrderedCache) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.purgeVisitorFunc != nil {
		for key, item := range c.items {
			c.purgeVisitorFunc(key, item.value)
		}
	}

	c.init()
}

func (c *SimpleOrderedCache) OrderedKeys() []interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.orderedKeys
}

func (c *SimpleOrderedCache) moveFront(key interface{}) (err error) {
	var try = 2
	c.mu.Lock()
	_, ok := c.items[key]
	if !ok {
		defer c.stats.IncrMissCount()
	}
	defer c.mu.Unlock()
	if ok {
		if len(c.orderedKeys) == 0 {
			return EmptyErr
		}
		//if key is first two item ,don't move it
		for i := 0; i < try && i < len(c.orderedKeys); i++ {
			head := c.orderedKeys[i]
			if head == key {
				return nil
			}
		}
		if len(c.orderedKeys) >= c.size {
			err = ReachedMaxSizeErr
		}
		arr := []interface{}{key}
		c.orderedKeys = append(arr, c.orderedKeys...)
		return err
	}
	return KeyNotFoundError
}

func (c *SimpleOrderedCache) MoveFront(key interface{}) error {
	if c.searchCmpFunc != nil {
		panic("don't call when items are sorted")
	}
	return c.moveFront(key)
}

func (c *SimpleOrderedCache) getALl() (keys []interface{}, values []interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var index []int
	for i, key := range c.orderedKeys {
		item, ok := c.items[key]
		if ok {
			if !item.IsExpired(nil) {
				v := item.value
				if c.deserializeFunc != nil {
					v, _ = c.deserializeFunc(key, v)
				}
				values = append(values, v)
				keys = append(keys, key)
				//c.stats.IncrHitCount()
			} else {
				index = append(index, i)
				log.Debugf("expired value will be removed %v %v", key, item.value)
				c.deleteVal(key)
			}
		} else {
			//c.stats.IncrMissCount()
			index = append(index, i)
		}
	}
	c.removeKeysByIndex(index)
	return keys, values
}

func (c *SimpleOrderedCache) getTop() (key interface{}, value interface{}, err error) {
	c.mu.Lock()
	var removedIndex []int
	for i, k := range c.orderedKeys {
		//key =  c.keyTypeFunction(k)
		key = k
		item, ok := c.items[key]
		if ok {
			value = item.value
			if item.IsExpired(nil) {
				removedIndex = append(removedIndex, i)
				log.Debugf("expired value will be removed %v %v", key, item.value)
				c.deleteVal(key)
			}
			break
		}
		removedIndex = append(removedIndex, i)
		log.Debugf("expired value will be removed %v ", key)
		c.deleteVal(key)
	}
	c.removeKeysByIndex(removedIndex)
	c.mu.Unlock()
	if value != nil {
		c.stats.IncrHitCount()
		return key, value, nil
	} else {
		c.stats.IncrMissCount()
		return nil, nil, EmptyErr
	}
}

func (c *SimpleOrderedCache) GetTop() (key interface{}, value interface{}, err error) {
	return c.getTop()
}

func (c *SimpleOrderedCache) deQueueBatch(count int) (keys []interface{}, values []interface{}, err error) {
	c.mu.Lock()
	var removedIndex []int
	current := 0
	all := false
	if count >= len(c.orderedKeys) {
		all = true
	}
	for i, key := range c.orderedKeys {
		item, ok := c.items[key]
		if ok {
			if current >= count {
				break
			}
			value := item.value
			values = append(values, value)
			keys = append(keys, key)
			current++
		}
		removedIndex = append(removedIndex, i)
		c.deleteVal(key)
	}
	if all {
		log.Debugf("init cache , removed all")
		c.init()
	} else {
		c.removeKeysByIndex(removedIndex)
	}
	c.mu.Unlock()
	if len(values) != 0 {
		c.stats.IncrHitCount()
		return keys, values, nil
	} else {
		c.stats.IncrMissCount()
		return nil, nil, EmptyErr
	}
}

func (c *SimpleOrderedCache) deQueue() (key interface{}, value interface{}, err error) {
	c.mu.Lock()
	var removedIndex []int
	for i, key := range c.orderedKeys {
		removedIndex = append(removedIndex, i)
		item, ok := c.items[key]
		if ok {
			value = item.value
			c.deleteVal(key)
			break
		}
		c.deleteVal(key)
	}
	c.removeKeysByIndex(removedIndex)
	c.mu.Unlock()
	if value != nil {
		c.stats.IncrHitCount()
		return key, value, nil
	} else {
		c.stats.IncrMissCount()
		return key, nil, EmptyErr
	}
}

func (c *SimpleOrderedCache) DeQueue() (key interface{}, value interface{}, err error) {
	return c.deQueue()
}

func (c *SimpleOrderedCache) DeQueueBatch(count int) (keys []interface{}, values []interface{}, err error) {
	return c.deQueueBatch(count)
}

func (c *SimpleOrderedCache) EnQueue(key interface{}, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.enQueue(key, value)
	return err
}

//EnQueueBatch if searchCmpFunc is not nil , keys are  required sorted
func (c *SimpleOrderedCache) EnQueueBatch(keys []interface{}, values []interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enQueueBatch(keys, values)
}

func (c *SimpleOrderedCache) PrependBatch(keys []interface{}, values []interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.addFrontBatch(keys, values)
}

func (c *SimpleOrderedCache) Prepend(key interface{}, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.addFront(key, value)
	return err
}

func (c *SimpleOrderedCache) RemoveExpired(allowFailCount int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.removeExpired(allowFailCount)
}

func (c *SimpleOrderedCache) Sort() {
	c.mu.Lock()
	defer c.mu.Unlock()
	getItem := func(key interface{}) (value interface{}, ok bool) {
		item, ok := c.items[key]
		if ok {
			value = item.value
			if item.IsExpired(nil) {
				log.Debugf("remove expired key %v, value %v", key, value)
				c.delete(key)
			}
		}
		return value, ok
	}
	var values []interface{}
	for key, item := range c.items {
		if item.IsExpired(nil) {
			c.delete(key)
		}
		values = append(values, item.value)
	}
	keys, _ := c.sortKeysFunc(c.orderedKeys, values, getItem)
	c.orderedKeys = keys
}

func (c *SimpleOrderedCache) removeExpired(allowFailCount int) error {
	var try int
	var removedIndex []int
	for i, k := range c.orderedKeys {
		//key :=  c.keyTypeFunction(k)
		key := k
		if try > allowFailCount {
			break
		}
		item, ok := c.items[key]
		if ok {
			if item.IsExpired(nil) {
				removedIndex = append(removedIndex, i)
				c.deleteVal(key)
				continue
			} else if c.expireFunction != nil {
				if c.expireFunction(key) {
					removedIndex = append(removedIndex, i)
					c.deleteVal(key)
					continue
				}
			}
			try++
		} else {
			removedIndex = append(removedIndex, i)
			c.deleteVal(key)
		}
	}
	c.removeKeysByIndex(removedIndex)
	return nil
}

//for test
func (c *SimpleOrderedCache) PrintValues(stamp int) {
	if stamp <= 0 {
		stamp = 0
	}
	keys, values := c.GetKeysAndValues()
	fmt.Println("total  , key, value", len(keys), len(values))
	for i, key := range keys {
		if i%stamp == 0 {
			if i >= len(values) {
				fmt.Println("key ", key)
			}
			fmt.Println("key  value ", key, values[i])
		}
	}
}
