package ffchan

import (
	"time"
	"github.com/sirupsen/logrus"
	"reflect"
)

type TimeoutSender struct {
	channel   interface{}
	val       interface{}
	groupName string
	timeout   time.Duration
	C         chan bool
}

func NewTimeoutSender(channel interface{}, val interface{}, groupName string, timeoutMs int) *TimeoutSender {
	t := &TimeoutSender{
		groupName: groupName,
		channel:   channel,
		timeout:   time.Duration(time.Millisecond * time.Duration(timeoutMs)),
		val:       val,
		C:         make(chan bool),
	}
	c := make(chan struct{})
	go func() {
		defer close(c)
		vChan := reflect.ValueOf(t.channel)
		vVal := reflect.ValueOf(t.val)
		vChan.Send(vVal)
	}()

	go func() {
		start := time.Now()
	loop:
		for {
			select {
			case <-c:
				t.C <- true
				break loop
			case <-time.After(t.timeout):
				logrus.WithField("chan", t.groupName).
					WithField("val", t.val).
					WithField("elapse", time.Now().Sub(start)).
					Warn("Timeout on channel writing. Potential block issue.")
				if t.timeout < time.Second {
					time.Sleep(time.Second)
				}
			}
		}
	}()

	return t
}
