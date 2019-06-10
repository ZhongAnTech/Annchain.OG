package gcache

import (
	"container/list"
	"fmt"
	"sort"
	"testing"
	"time"
)

func loader(key interface{}) (interface{}, error) {
	return fmt.Sprintf("valueFor%s", key), nil
}

func testSetCache(t *testing.T, gc Cache, numbers int) {
	for i := 0; i < numbers; i++ {
		key := fmt.Sprintf("Key-%d", i)
		value, err := loader(key)
		if err != nil {
			t.Error(err)
			return
		}
		gc.Set(key, value)
	}
}

func testGetCache(t *testing.T, gc Cache, numbers int) {
	for i := 0; i < numbers; i++ {
		key := fmt.Sprintf("Key-%d", i)
		v, err := gc.Get(key)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		expectedV, _ := loader(key)
		if v != expectedV {
			t.Errorf("Expected value is %v, not %v", expectedV, v)
		}
	}
}

func testGetIFPresent(t *testing.T, evT string) {
	cache :=
		New(8).
			EvictType(evT).
			LoaderFunc(
				func(key interface{}) (interface{}, error) {
					return "value", nil
				}).
			Build()

	v, err := cache.GetIFPresent("key")
	if err != KeyNotFoundError {
		t.Errorf("err should not be %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	v, err = cache.GetIFPresent("key")
	if err != nil {
		t.Errorf("err should not be %v", err)
	}
	if v != "value" {
		t.Errorf("v should not be %v", v)
	}
}

func testGetALL(t *testing.T, evT string) {
	size := 8
	cache :=
		New(size).
			Expiration(time.Millisecond).
			EvictType(evT).
			Build()
	for i := 0; i < size; i++ {
		cache.Set(i, i*i)
	}
	m := cache.GetALL()
	for i := 0; i < size; i++ {
		v, ok := m[i]
		if !ok {
			t.Errorf("m should contain %v", i)
			continue
		}
		if v.(int) != i*i {
			t.Errorf("%v != %v", v, i*i)
			continue
		}
	}
	time.Sleep(time.Millisecond + time.Microsecond*100)

	cache.Set(size, size*size)
	m = cache.GetALL()
	if len(m) != 1 {
		t.Errorf("%v != %v", len(m), 1)
	}
	if _, ok := m[size]; !ok {
		t.Errorf("%v should contains key '%v'", m, size)
	}
}

func TestAppend1(t *testing.T) {
	var sli []int
	for j := 1; j < 300000; j++ {
		sli = append(sli, j)
	}
	start := time.Now()
	for i := 0; i < len(sli); {
		v := sli[i]
		if v%3 == 0 {
			//fmt.Println(v)
			sli = append(sli[:i], sli[i+1:]...)
			//fmt.Println(sli)
		} else {
			i++
		}
	}
	//=== RUN   TestAppend1
	//used time 23.945s 200000
	//--- PASS: TestAppend1 (24.09s)
	//PASS

	//Process finished with exit code 0
	fmt.Println("used time", time.Now().Sub(start), len(sli))

}

func TestAppend2(t *testing.T) {
	start := time.Now()
	var sli []int
	for j := 1; j < 300000; j++ {
		sli = append(sli, j)
	}
	var result []int
	for i := 0; i < len(sli); i++ {
		v := sli[i]
		if v%3 == 0 {
			//fmt.Println(v)

			//fmt.Println(sli)
		} else {
			result = append(result, v)
		}
	}
	sli = result
	fmt.Println("used time", time.Now().Sub(start), len(result), len(sli))
	//=== RUN   TestAppend2
	//used time 25ms 200000 200000
	//--- PASS: TestAppend2 (0.03s)
	//PASS
	//
	//Process finished with exit code 0

}

func TestMoveFront(t *testing.T) {
	var arr = []int{2, 3, 4, 5, 6, 7, 8, 9}
	fmt.Println(arr)
	arr = append([]int{1}, arr...)
	fmt.Println(arr)
}

func TestSliceInsert(t *testing.T) {
	var arr []string
	start := time.Now()
	for i := 3; i < 10000000; i++ {
		arr = append(arr, fmt.Sprintf("i%di", i))
	}
	fmt.Println("use time", time.Now().Sub(start), len(arr))
	start = time.Now()
	arr = append(arr, "10")
	fmt.Println("use time", time.Now().Sub(start), len(arr))
	start = time.Now()
	arr = append([]string{"00"}, arr...)
	fmt.Println("use time", time.Now().Sub(start), len(arr))
	start = time.Now()
	arrCopy := make([]string, len(arr)+1)
	arrCopy[0] = "11"
	copy(arrCopy[1:], arr)
	arr = arrCopy
	fmt.Println("use time", time.Now().Sub(start), len(arr))
}

func TestListInsert(t *testing.T) {
	l := list.New()
	start := time.Now()
	for i := 3; i < 10000000; i++ {
		//arr = append(arr,fmt.Sprintf("i%di",i))
		l.PushBack(fmt.Sprintf("i%di", i))
	}
	fmt.Println("use time", time.Now().Sub(start), l.Len())
	start = time.Now()
	l.PushBack("10")
	fr := l.Front()
	fmt.Println("use time", time.Now().Sub(start), l.Len(), "front", fr)
	start = time.Now()
	f := l.Front()
	l.InsertBefore("11", f)
	fr = l.Front()
	fmt.Println("use time", time.Now().Sub(start), l.Len(), "front ", fr)
}

func TestBinSearch(t *testing.T) {
	var s []int
	for i := 0; i < 40960; i++ {
		s = append(s, i)
	}
	m := 2089
	f := func(i int) bool {
		return s[i] >= m
	}
	total := 0
	n := len(s)
	// Define f(-1) == false and f(n) == true.
	// Invariant: f(i-1) == false, f(j) == true.
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		if !f(h) {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
		total++
	}
	fmt.Println(total)
	// i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.
	fmt.Println(i)
}

func TestArray(t *testing.T) {
	var s []string
	start := time.Now()
	for i := 0; i < 5000000; i = i + 2 {
		s = append(s, fmt.Sprintf("%d", i))
	}
	fmt.Println("used for gen data ", time.Now().Sub(start), len(s))
	//insert 100000
	key := 1000001
	val := fmt.Sprintf("%d", key)
	var total = 0
	n := len(s)
	f := func(i int) bool {
		total++
		if s[i] >= val {
			return true
		}
		return false
	}
	start = time.Now()
	i := sort.Search(len(s), f)
	fmt.Println("used for search", time.Now().Sub(start), len(s), i)
	s1 := append(s[0:i], val)
	s = append(s1, s[i:]...)
	fmt.Println(i, total, s[i-1], s[i], s[i+1])
	fmt.Println("used for insert", time.Now().Sub(start), len(s))

	start = time.Now()
	n = len(s)
	s = append(s, "0")
	for i := n; i > 0 && val < s[i-1]; i-- {
		s[i] = s[i-1]
	}
	s[i] = val
	fmt.Println(i, val, s[i-1], s[i+1])
	fmt.Println("used for copy ", time.Now().Sub(start), len(s))

}

func TestSearch(t *testing.T) {
	a := []int{1, 3, 6, 8, 10, 15, 21, 28, 36, 45, 55}
	x := 9

	i := Search(len(a), func(i int) bool { return a[i] >= x })
	if i < len(a) && a[i] == x {
		fmt.Printf("found %d at index %d in %v\n", x, i, a)
	} else {
		fmt.Printf("%d not found in %v i: %d\n", x, a, i)
	}
	// Output:
	// found 6 at index 2 in [1 3 6 10 15 21 28 36 45 55]

	var s []int
	for i := 0; i < 5000000; i = i + 2 {
		s = append(s, i)
	}
	target := 10003
	i = Search(len(s), func(i int) bool { return s[i] >= target })
	if i < len(a) && s[i] == target {
		fmt.Printf("found %d at index %d in %v\n", target, i, len(s))
	} else {
		fmt.Printf("%d not found in %v at %d  %d\n ", target, len(s), i, s[i])
	}
}

func TestSliceInsert2(t *testing.T) {
	type Slice []interface{}
	f := func() Slice {
		s := []interface{}{}
		for i := 0; i < 1000000; i++ {
			s = append(s, i*2)
		}
		return s
	}
	var slices []Slice
	initSlice := f()
	fmt.Println(len(initSlice))
	for i := 0; i < 10; i++ {
		var tmp = make(Slice, len(initSlice))
		copy(tmp, initSlice)
		slices = append(slices, tmp)
	}
	start := time.Now()
	s := SliceInsert(slices[0], 3, 9)
	fmt.Println("insert 1 used ", time.Now().Sub(start), len(s))
	start = time.Now()
	s = SliceInsert2(slices[1], 3, 9)
	fmt.Println("insert 2  used ", time.Now().Sub(start), len(s))
	start = time.Now()
	s = SliceInsert3(slices[2], 3, 9)
	fmt.Println("insert 3  used ", time.Now().Sub(start), len(s))
	start = time.Now()
	s = SliceInsert4(slices[3], 3, 9)
	fmt.Println("insert 4  used ", time.Now().Sub(start), len(s))
	start = time.Now()
	s = SliceInsert(slices[4], 300000, 9)
	fmt.Println("insert 1 used ", time.Now().Sub(start), len(s))
	start = time.Now()
	s = SliceInsert2(slices[5], 300000, 9)
	fmt.Println("insert 2  used ", time.Now().Sub(start), len(s))
	start = time.Now()
	s = SliceInsert3(slices[6], 300000, 9)
	fmt.Println("insert 3  used ", time.Now().Sub(start), len(s))
	s = SliceInsert3(slices[7], 300000, 9)
	fmt.Println("insert 4  used ", time.Now().Sub(start), len(s))
	//1000000
	//insert 1 used  24ms 1000001
	//insert 2  used  512ms 1000001
	//insert 3  used  46ms 1000001
	//insert 4  used  33ms 1000001
	//insert 1 used  14ms 1000001
	//insert 2  used  27ms 1000001
	//insert 3  used  76ms 1000001
	//insert 4  used  420ms 1000001
}

func TestSearch2(t *testing.T) {
	var arrs = []int{1, 2, 3, 4, 5, 6, 6, 6, 7, 7, 7, 8, 8, 8, 9, 9, 9, 9, 10}
	x := 10
	f := func(i int) bool {
		if arrs[i] >= x {
			return true
		}
		return false
	}
	i := Search(len(arrs), f)
	fmt.Println(i, arrs[i], x, arrs[i-1], len(arrs))

}
