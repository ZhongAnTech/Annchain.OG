package tendermint

import (
	"testing"
	"time"
	"fmt"
)

func callback(w WaiterContext) {
	fmt.Println(w.(*TendermintContext).HeightRound.Height)
}

func TestWait(t *testing.T) {
	c := make(chan *WaiterRequest)
	w := NewWaiter(c)
	go w.StartEventLoop()
	w.UpdateRequest(&WaiterRequest{
		WaitTime:        time.Second * 5,
		TimeoutCallback: callback,
		Context: &TendermintContext{
			HeightRound:HeightRound{
				Height: 2,
				Round:  3,
			},

		},
	})
	w.UpdateContext(&TendermintContext{
		HeightRound:HeightRound{
			Height: 2,
			Round:  3,
		},
	})

	time.Sleep(time.Second * 100)
}
