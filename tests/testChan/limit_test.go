package testChan

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestLimit(t *testing.T) {
	a := app{
		ch:   make(chan int, 3),
		quit: make(chan bool),
	}
	//go a.loop()

	for i := 0; i < 10; i++ {
		fmt.Println("before write  ", i, time.Now())
		a.process(i)
		fmt.Println("after write ", i, time.Now())
	}

}

type app struct {
	ch   chan int
	quit chan bool
}

func (a *app) loop() {

	for {
		select {
		case i := <-a.ch:
			fmt.Println("start ", i, time.Now())
			time.Sleep(time.Millisecond * 500)
			fmt.Println("stop ", i, time.Now())
		case <-a.quit:
			fmt.Println("exit")
			return

		}
	}
}

func (a *app) process(i int) {
	a.ch <- i
	fmt.Println(runtime.NumGoroutine(), "goroutine num")
	go func() {
		fmt.Println("start ", i, time.Now())
		time.Sleep(time.Millisecond * 1000)
		fmt.Println("stop ", i, time.Now())
		<-a.ch
	}()

}
