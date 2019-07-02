package goroutine

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"runtime/debug"
	"sync/atomic"
	"time"
)

var globalGoRoutineNum int32

var calculateGoroutineNum = true

func GetGoRoutineNum() int32 {
	return atomic.LoadInt32(&globalGoRoutineNum)
}

func New(function func()) {
	//todo we can handle goroutine num here
	if calculateGoroutineNum {
		atomic.AddInt32(&globalGoRoutineNum, 1)
	}
	go func() {
		if calculateGoroutineNum {
			defer atomic.AddInt32(&globalGoRoutineNum, -1)
		}
		defer DumpStack(true)
		function()
	}()
}

func WithRecover(handler func()) {
	//todo we can handle goroutine num here
	go func() {
		defer DumpStack(false)
		handler()
	}()
}

func DumpStack(exitIFPanic bool) {
	if err := recover(); err != nil {
		fmt.Println("goroutine num ",calculateGoroutineNum)
		logrus.WithField("obj", err).Error("Fatal error occurred. Program will exit")
		var buf bytes.Buffer
		stack := debug.Stack()
		buf.WriteString(fmt.Sprintf("Panic: %v\n", err))
		buf.Write(stack)
		dumpName := "dump_" + time.Now().Format("20060102-150405")
		nerr := ioutil.WriteFile(dumpName, buf.Bytes(), 0644)
		if nerr != nil {
			fmt.Println("write dump file error", nerr)
			fmt.Println(buf.String())
		}
		logrus.Errorf("panic %v ", buf.String())
		if exitIFPanic {
			panic(err)
		}
	}
}
