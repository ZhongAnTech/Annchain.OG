package testChan

import (
	"fmt"
	"testing"
)

type s struct {
	ch      chan bool
	stopped bool
}

func TestA(t *testing.T) {

	var a = make(chan bool)
	var pa = &a
	var b chan bool
	var pb = &b
	fmt.Println(a, b, pa, pb)
	b = a
	pb = pa
	fmt.Println(a, b, pa, pb)
	a = nil
	pa = nil
	fmt.Println(a, b, pa, pb)

}

func TestB(t *testing.T) {
	var a = s{
		ch:      make(chan bool),
		stopped: false,
	}
	var c = &a
	var b = a
	fmt.Println(a, c, b)
	a.stopped = true
	fmt.Println(a, c, b)
}
