package wserver

import (
	"fmt"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	addr := ":12345"
	srv := NewServer(addr)
	go func() {
		//time.Sleep(time.Second * time.Duration(5))
		for i := 0; i < 10000; i++ {
			srv.Push(EVENT_NEW_UNIT, fmt.Sprintf("msg %d", i))
			time.Sleep(time.Millisecond * time.Duration(500))
		}
	}()
	srv.Serve()

	time.Sleep(time.Second * 60)
}
