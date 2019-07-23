package gcache

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"sort"
	"testing"
	"time"
)

func newtestSimploOrderCache(serachFunc SearchCompareFunction) *SimpleOrderedCache {
	DebugMode = true
	logrus.SetLevel(logrus.TraceLevel)
	c := New(100000).Simple().Expiration(time.Second * 10).SearchCompareFunction(serachFunc).BuildOrderedCache()
	return c.(*SimpleOrderedCache)

}

func TestSimpleOrderedCache_Remove(t *testing.T) {
	c := newtestSimploOrderCache(nil)
	for i := 0; i < 1000; i++ {
		c.EnQueue(i, fmt.Sprintf("%d", i))
	}
	t.Log(c.Len(), c.orderedKeys)
	//index := []int{0}ls

	//c.removeKeysByIndex(index)
	t.Log(len(c.orderedKeys), c.orderedKeys)
	total := c.Len()
	for i := 0; i < total; i++ {
		c.DeQueue()
	}
	t.Log(c.Len(), c.orderedKeys)
}

func TestSimpleOrderedCache_EnQueueBatch(t *testing.T) {
	c := newtestSimploOrderCache(nil)
	for i := 0; i < 100; i++ {
		c.EnQueue(i, fmt.Sprintf("%d", i))
	}
	var keys []interface{}
	var values []interface{}
	for i := 0; i < 100; i++ {
		keys = append(keys, 200+i)
		values = append(values, fmt.Sprintf("hi%d", i))
	}
	t.Log(len(keys), len(values))
	c.EnQueueBatch(keys, values)
	t.Log(c.Len(), c.orderedKeys)
	total := c.Len()
	for i := 0; i < total/2; i++ {
		c.DeQueue()
	}
	t.Log(c.Len(), c.orderedKeys)
	keys, values, err := c.DeQueueBatch(40)
	t.Log(len(keys), len(values), err)
	t.Log(len(c.orderedKeys), c.orderedKeys)
	var itemKeys []int
	for k := range c.items {
		itemKeys = append(itemKeys, k.(int))
	}
	sort.Ints(itemKeys)
	t.Log(len(itemKeys), itemKeys)
	t.Log(c.Len())
	val, err := c.GetIFPresent(256)
	t.Log(val, err)
}

func TestRemoveByIndex(t *testing.T) {
	c := newtestSimploOrderCache(nil)
	var index []int
	for i := 0; i < 100; i++ {
		c.orderedKeys = append(c.orderedKeys, i)
		index = append(index, i)
	}
	c.removeKeysByIndex(index)
	if len(c.orderedKeys) != 0 {
		t.Fatalf("need 0, got %d %v", c.Len(), c.orderedKeys)
	}
}

func TestSimpleOrderedCache_MoveFront(t *testing.T) {
	c := newtestSimploOrderCache(nil)
	for i := 0; i < 100; i++ {
		val := fmt.Sprintf("I%d", i)
		c.EnQueue(i, val)
	}
	err := c.MoveFront(1)
	fmt.Println(err, c.orderedKeys)
	err = c.MoveFront(10)
	fmt.Println(err, c.orderedKeys)
}

func TestSimpleOrderedCache_GetKeysAndValues(t *testing.T) {
	c := newtestSimploOrderCache(nil)
	for i := 0; i < 20; i++ {
		val := fmt.Sprintf("I%d", i)
		c.EnQueue(i, val)
	}
	keys, values := c.GetKeysAndValues()
	fmt.Println(keys)
	fmt.Println(values)
}

func TestSimpleOrderedCache_AddFront(t *testing.T) {
	c := newtestSimploOrderCache(nil)
	for i := 0; i < 20; i++ {
		val := fmt.Sprintf("I%d", i)
		c.EnQueue(i, val)
	}
	c.Prepend(25, "i25")
	keys, values := c.GetKeysAndValues()
	fmt.Println(keys)
	fmt.Println(values)
}

func TestSimpleOrderedCache_AddFrontBatch(t *testing.T) {
	c := newtestSimploOrderCache(nil)
	for i := 0; i < 20; i++ {
		val := fmt.Sprintf("I%d", i)
		c.EnQueue(i, val)
	}
	c.PrependBatch([]interface{}{30, 31, 32, 33}, []interface{}{"30", "31", "32", "33"})
	keys, values := c.GetKeysAndValues()
	fmt.Println(keys)
	fmt.Println(values)
}

func cmpInt(x interface{}, y interface{}) int {
	a := x.(int)
	b := y.(int)
	if a > b {
		return 1
	} else if a < b {
		return -1
	}
	return 0
}

func TestSimpleCache_EnqueueWithOrder(t *testing.T) {
	fmt.Println(cmpInt(5, 3))
	c := newtestSimploOrderCache(cmpInt)
	for i := 0; i < 10; i++ {
		val := 3*i + 1
		//insert at the end
		c.EnQueue(i, val)
	}
	c.PrintValues(1)
	//insert in the middle
	c.EnQueue(21, 23)
	c.PrintValues(1)
	//insert at the front
	c.EnQueue(-1, -3)
	c.PrintValues(1)

}

func TestSimpleCache_BatchEnqueueWithOrder(t *testing.T) {

	fmt.Println(cmpInt(5, 3))
	c := newtestSimploOrderCache(cmpInt)
	for i := 0; i < 20; i = i + 2 {
		val := 3*i + 1
		//insert at the end
		c.EnQueue(i, val)
	}
	c.PrintValues(1)
	//insert at  the front
	c.EnQueueBatch([]interface{}{101, 102, 103}, []interface{}{-4*3 + 1, -2*3 + 1, -1*3 + 1})
	c.PrintValues(1)
	//insert at the end
	c.EnQueueBatch([]interface{}{80, 81, 82}, []interface{}{102*3 + 1, 103*3 + 1, 104*3 + 1})
	c.PrintValues(1)

	//insert in the middle
	c.EnQueueBatch([]interface{}{30, 31, 32, 33, 34, 35, 36}, []interface{}{8*3 + 1, 12*3 + 1, 12*3 + 1, 13 * 3, 15*3 + 1, 15*3 + 1, 17*3 + 1})
	c.PrintValues(1)

}

func TestSimpleCache_BatchAddFrontWithOrder(t *testing.T) {
	fmt.Println(cmpInt(5, 3))
	c := newtestSimploOrderCache(cmpInt)
	for i := 0; i < 20; i = i + 2 {
		val := 3*i + 1
		//insert at the end
		c.EnQueue(i, val)
	}
	c.PrintValues(1)
	c.orderedKeys = append(c.orderedKeys, 54, 55)
	SliceInsert(c.orderedKeys, 0, 56)
	SliceInsert(c.orderedKeys, 0, 57)
	//insert at  the front
	c.PrependBatch([]interface{}{101, 102, 103}, []interface{}{-4*3 + 1, -1*3 + 1, 23 - 2*3 + 1})
	c.PrintValues(1)
	c.orderedKeys = append(c.orderedKeys, 58, 59)
	SliceInsert(c.orderedKeys, 0, 567)
	SliceInsert(c.orderedKeys, 0, 577)
	//insert at the end
	c.EnQueueBatch([]interface{}{80, 81, 82}, []interface{}{-102 + 103*3 + 1, 102*3 + 1, 104*3 + 1})
	c.PrintValues(1)

	//insert in the middle
	c.orderedKeys = append(c.orderedKeys, 50, 51)
	SliceInsert(c.orderedKeys, 0, 52)
	SliceInsert(c.orderedKeys, 0, 53)
	c.PrependBatch([]interface{}{30, 31, 32, 33, 34, 35, 36}, []interface{}{6 - 13*3, 8*3 + 1, 12*3 + 1, 12*3 + 1, 15*3 + 1, 15*3 + 1, 17*3 + 1})
	c.PrintValues(1)
	c.EnQueueBatch([]interface{}{40}, []interface{}{30})
	c.PrintValues(1)
}

func TestSimpleCache_WithOrder(t *testing.T) {
	c := newtestSimploOrderCache(cmpInt)
	for i := 10000; i < 30000; i = i + 2 {
		val := 3*i + 1
		//insert at the end
		c.EnQueue(i, val)
	}
	var smallSlicekey []interface{}
	var smallSliceVal []interface{}
	for i := 2800; i < 4100; i = i + 2 {
		smallSlicekey = append(smallSlicekey, i)
		var val int
		if i%5 == 0 {
			val = i * 10
		} else {
			val = i*10 + 1
		}
		if i > 4000 {
			val = 60000 + i
		}
		if i > 4050 {
			val = 80000 + i*11
		}
		//insert at the end
		smallSliceVal = append(smallSliceVal, val)
	}
	c.PrintValues(400)
	start := time.Now()
	c.PrependBatch(smallSlicekey, smallSliceVal)
	fmt.Println("ued time ", time.Now().Sub(start))
	c.PrintValues(300)
}

func TestSimpleCache_Remove(t *testing.T) {
	c := newtestSimploOrderCache(cmpInt)
	var oldValue int
	for i := 10000; i < 40000; i = i + 2 {
		var value int
		if i%8 == 0 {
			value = 3*i + 1
			oldValue = value
		} else {
			value = oldValue
		}
		//insert at the end
		c.EnQueue(i, value)
		if i <= 10000+30 {
			fmt.Print(" i ", i, value, " v ")
			if i == 10000+30 {
				fmt.Println("")
			}
		}
	}
	c.PrintValues(900)
	start := time.Now()
	c.Remove(23204)
	fmt.Println("ued time for sorted remove", time.Now().Sub(start))
	c.PrintValues(800)
	start = time.Now()
	c.searchCmpFunc = nil
	c.Remove(23206)
	fmt.Println("ued time for remove", time.Now().Sub(start))
	c.PrintValues(800)
}

func TestSimpleCache_Sort(t *testing.T) {
	type element struct {
		height int
		weight int
	}
	f := func(x interface{}, y interface{}) int {
		e := x.(element)
		e2 := y.(element)
		if e.weight > e2.weight {
			return 1
		} else if e.weight < e2.weight {
			return -1
		}
		return 0
	}
	c := newtestSimploOrderCache(f)
	for i := 0; i < 100; i++ {
		var e element
		if i%40 == 0 {
			e.height = i / 10
			e.weight = int(i%100) + e.height
		} else {
			e.height = i / 10
			e.weight = int(i%10) + e.height
		}
		fmt.Print(e, "", i)
		err := c.EnQueue(i, e)
		if err != nil {
			panic(err)
		}
	}
	c.PrintValues(1)
	//fmt.Pr
}

func TestRemoveLastItem(t *testing.T) {
	c := newtestSimploOrderCache(cmpInt)
	var oldVal int
	for i := 0; i < 10; i++ {
		var val int
		if i%3 == 0 {
			val = i * 2
			oldVal = val
		} else {
			val = oldVal
		}
		c.EnQueue(i, val)
	}
	c.PrintValues(1)
	fmt.Println(c.Get(9))
	c.Remove(9)
	fmt.Println(c.Get(9))
	c.PrintValues(1)
}

func TestSimpleCache_BatchAddFront(t *testing.T) {
	fmt.Println(cmpInt(5, 3))
	c := newtestSimploOrderCache(cmpInt)
	for i := 0; i < 30; i = i + 2 {
		val := 3*i + 1
		//insert at the end
		c.EnQueue(i, val)
	}
	c.PrintValues(1)
	//insert at  the front
	c.PrependBatch([]interface{}{101, 102, 103}, []interface{}{-4*3 + 1, -1*3 + 1, 23 - 2*3 + 1})
	c.PrintValues(1)
	//insert at the end
	c.EnQueueBatch([]interface{}{80, 81, 82}, []interface{}{-102 + 103*3 + 1, 102*3 + 1, 104*3 + 1})
	c.PrintValues(1)

	//insert in the middle

	c.PrependBatch([]interface{}{30, 31, 32, 33, 34, 35, 36}, []interface{}{6 - 13*3, 8*3 + 1, 12*3 + 1, 12*3 + 1, 15*3 + 1, 15*3 + 1, 17*3 + 1})
	c.PrintValues(1)
}
