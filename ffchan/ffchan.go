// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package ffchan

import (
	"github.com/sirupsen/logrus"
	"reflect"
	"time"
)

type TimeoutSender struct {
	channel   interface{}
	val       interface{}
	groupName string
	timeout   time.Duration
	C         chan bool
}

func NewTimeoutSenderShort(channel interface{}, val interface{}, groupName string) *TimeoutSender {
	return NewTimeoutSender(channel, val, groupName, 5000)
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
