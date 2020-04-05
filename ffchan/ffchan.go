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
	"fmt"
	"github.com/annchain/OG/common/goroutine"
	"github.com/sirupsen/logrus"
	"math/rand"
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
	ri := rand.Int()
	logrus.WithField("groupName", groupName).WithField("rand", ri).Trace("ffchan received a msg to send")
	t := &TimeoutSender{
		groupName: groupName,
		channel:   channel,
		timeout:   time.Duration(time.Millisecond * time.Duration(timeoutMs)),
		val:       val,
		C:         make(chan bool),
	}
	c := make(chan struct{})
	goroutine.New(func() {
		defer close(c)
		vChan := reflect.ValueOf(t.channel)
		vVal := reflect.ValueOf(t.val)
		vChan.Send(vVal)
	})

	goroutine.New(func() {
		start := time.Now()
	loop:
		for {
			select {
			case <-c:
				select {
				case t.C <- true:
					logrus.WithField("groupName", groupName).WithField("rand", ri).Trace("message now in channel")
				case <-time.After(t.timeout):
					logrus.WithField("groupName", groupName).WithField("rand", ri).Warn("ffchan cannot write to this channel. Are you missing a consume operation (<- xx.C)?")
				}
				break loop
			case <-time.After(t.timeout):
				logrus.WithField("groupName", t.groupName).
					WithField("rand", ri).
					WithField("val", t.val).
					WithField("elapse", time.Now().Sub(start)).
					Warn("Timeout on channel writing. Potential block issue.")
				fmt.Println("Timeout on channel writing: " + groupName)
				if t.timeout < time.Second {
					time.Sleep(time.Second)
				}
			}
		}
	})

	return t
}
