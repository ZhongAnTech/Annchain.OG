package tendermint

import (
	"testing"
	"time"
	"fmt"
)

func callback(w WaiterContext) {
	fmt.Println(w.(*TendermintContext).Height)
}

func TestWait(t *testing.T) {
	w := NewWaiter()
	go w.StartEventLoop()
	w.UpdateRequest(&WaiterRequest{
		WaitTime:        time.Second * 5,
		TimeoutCallback: callback,
		Context: &TendermintContext{
			Height: 2,
			Round:  3,
		},
	})
	w.UpdateContext(&TendermintContext{
		Height: 3,
		Round:  2,
	})

	time.Sleep(time.Second * 100)
}
