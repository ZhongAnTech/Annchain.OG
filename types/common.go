package types

import (
	"time"
)

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func ChannelHandler(timer *time.Timer, receiver chan interface{}, data interface{}) {
	for {
		if !timer.Stop() {
			<-timer.C
		}
		timer.Reset(time.Second * 10)
		select {
		case <-timer.C:
			// log.Warn("timeout on channel writing: batch confirmed")
		case receiver <- data:
			break
		}
	}
}
